package buildpack

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/env"
	"github.com/buildpacks/lifecycle/internal/encoding"
	"github.com/buildpacks/lifecycle/internal/fsutil"
	"github.com/buildpacks/lifecycle/launch"
	"github.com/buildpacks/lifecycle/layers"
	"github.com/buildpacks/lifecycle/log"
)

const (
	EnvLayersDir  = "CNB_LAYERS_DIR"
	EnvBpPlanPath = "CNB_BP_PLAN_PATH"

	EnvOutputDir = "CNB_OUTPUT_DIR"
)

type BuildEnv interface { // TODO: move into BuildConfig
	AddRootDir(baseDir string) error
	AddEnvDir(envDir string, defaultAction env.ActionType) error
	WithPlatform(platformDir string) ([]string, error)
	List() []string
}

type BuildConfig struct {
	AppDir          string
	OutputParentDir string // TODO: rename
	PlatformDir     string
	Out             io.Writer
	Err             io.Writer
	Logger          log.Logger
}

//go:generate mockgen -package testmock -destination ../testmock/build_executor.go github.com/buildpacks/lifecycle/buildpack BuildExecutor
type BuildExecutor interface {
	Build(d *BpDescriptor, plan Plan, config BuildConfig, buildEnv BuildEnv) (BuildResult, error)
}

type DefaultBuildExecutor struct{}

func (e *DefaultBuildExecutor) Build(d *BpDescriptor, plan Plan, config BuildConfig, buildEnv BuildEnv) (BuildResult, error) {
	if api.MustParse(d.WithAPI).Equal(api.MustParse("0.2")) {
		config.Logger.Debug("Updating plan entries")
		for i := range plan.Entries {
			plan.Entries[i].convertMetadataToVersion()
		}
	}

	config.Logger.Debug("Creating plan directory")
	planDir, err := ioutil.TempDir("", launch.EscapeID(d.Buildpack.ID)+"-")
	if err != nil {
		return BuildResult{}, err
	}
	defer os.RemoveAll(planDir)

	config.Logger.Debug("Preparing paths")
	moduleOutputDir, planPath, err := prepareBuildInputPaths(d.Buildpack.ID, plan, config.OutputParentDir, planDir)
	if err != nil {
		return BuildResult{}, err
	}

	config.Logger.Debug("Running build command")
	if err := d.runCmd(moduleOutputDir, planPath, config, buildEnv); err != nil {
		return BuildResult{}, err
	}

	config.Logger.Debug("Processing layers")
	createdLayers, err := d.processLayers(moduleOutputDir, config.Logger)
	if err != nil {
		return BuildResult{}, err
	}

	config.Logger.Debug("Updating environment")
	if err := d.setupEnv(createdLayers, buildEnv); err != nil {
		return BuildResult{}, err
	}

	config.Logger.Debug("Reading output files")
	return d.readOutputFilesBp(moduleOutputDir, planPath, plan, createdLayers, config.Logger)
}

func prepareBuildInputPaths(moduleID string, plan Plan, outputParentDir, parentPlanDir string) (string, string, error) {
	moduleDirName := launch.EscapeID(moduleID) // TODO: this logic should eventually move to the platform package

	// Create e.g., <layers>/<buildpack-id> or <output>/<extension-id>
	moduleOutputDir := filepath.Join(outputParentDir, moduleDirName)
	if err := os.MkdirAll(moduleOutputDir, 0777); err != nil {
		return "", "", err
	}

	// Create Buildpack Plan
	childPlanDir := filepath.Join(parentPlanDir, moduleDirName) // TODO: it's unclear if this child directory is necessary; consider removing
	if err := os.MkdirAll(childPlanDir, 0777); err != nil {
		return "", "", err
	}
	planPath := filepath.Join(childPlanDir, "plan.toml")
	if err := encoding.WriteTOML(planPath, plan); err != nil {
		return "", "", err
	}

	return moduleOutputDir, planPath, nil
}

func (d *BpDescriptor) runCmd(moduleOutputDir, planPath string, config BuildConfig, buildEnv BuildEnv) error {
	cmd := exec.Command(
		filepath.Join(d.WithRootDir, "bin", "build"),
		moduleOutputDir,
		config.PlatformDir,
		planPath,
	) // #nosec G204
	cmd.Dir = config.AppDir
	cmd.Stdout = config.Out
	cmd.Stderr = config.Err

	var err error
	if d.Buildpack.ClearEnv {
		cmd.Env = buildEnv.List()
	} else {
		cmd.Env, err = buildEnv.WithPlatform(config.PlatformDir)
		if err != nil {
			return err
		}
	}
	cmd.Env = append(cmd.Env, EnvBuildpackDir+"="+d.WithRootDir)
	if api.MustParse(d.WithAPI).AtLeast("0.8") {
		cmd.Env = append(cmd.Env,
			EnvPlatformDir+"="+config.PlatformDir,
			EnvBpPlanPath+"="+planPath,
			EnvLayersDir+"="+moduleOutputDir,
		)
	}

	if err := cmd.Run(); err != nil {
		return NewError(err, ErrTypeBuildpack)
	}
	return nil
}

func (d *BpDescriptor) processLayers(layersDir string, logger log.Logger) (map[string]LayerMetadataFile, error) {
	if api.MustParse(d.WithAPI).LessThan("0.6") {
		return eachLayer(layersDir, d.WithAPI, func(path, buildpackAPI string) (LayerMetadataFile, error) {
			layerMetadataFile, msg, err := DecodeLayerMetadataFile(path+".toml", buildpackAPI)
			if err != nil {
				return LayerMetadataFile{}, err
			}
			if msg != "" {
				logger.Warn(msg)
			}
			return layerMetadataFile, nil
		})
	}
	return eachLayer(layersDir, d.WithAPI, func(path, buildpackAPI string) (LayerMetadataFile, error) {
		layerMetadataFile, msg, err := DecodeLayerMetadataFile(path+".toml", buildpackAPI)
		if err != nil {
			return LayerMetadataFile{}, err
		}
		if msg != "" {
			return LayerMetadataFile{}, errors.New(msg)
		}
		if err := renameLayerDirIfNeeded(layerMetadataFile, path); err != nil {
			return LayerMetadataFile{}, err
		}
		return layerMetadataFile, nil
	})
}

func eachLayer(bpLayersDir, buildpackAPI string, fn func(path, api string) (LayerMetadataFile, error)) (map[string]LayerMetadataFile, error) {
	files, err := ioutil.ReadDir(bpLayersDir)
	if os.IsNotExist(err) {
		return map[string]LayerMetadataFile{}, nil
	} else if err != nil {
		return map[string]LayerMetadataFile{}, err
	}
	bpLayers := map[string]LayerMetadataFile{}
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".toml") {
			continue
		}
		path := filepath.Join(bpLayersDir, strings.TrimSuffix(f.Name(), ".toml"))
		layerMetadataFile, err := fn(path, buildpackAPI)
		if err != nil {
			return map[string]LayerMetadataFile{}, err
		}
		bpLayers[path] = layerMetadataFile
	}
	return bpLayers, nil
}

func renameLayerDirIfNeeded(layerMetadataFile LayerMetadataFile, layerDir string) error {
	// rename <layers>/<layer> to <layers>/<layer>.ignore if buildpack API >= 0.6 and all the types flags are set to false
	if !layerMetadataFile.Launch && !layerMetadataFile.Cache && !layerMetadataFile.Build {
		if err := fsutil.RenameWithWindowsFallback(layerDir, layerDir+".ignore"); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (d *BpDescriptor) setupEnv(createdLayers map[string]LayerMetadataFile, buildEnv BuildEnv) error {
	bpAPI := api.MustParse(d.WithAPI)
	for path, layerMetadataFile := range createdLayers {
		if !layerMetadataFile.Build {
			continue
		}
		if err := buildEnv.AddRootDir(path); err != nil {
			return err
		}
		if err := buildEnv.AddEnvDir(filepath.Join(path, "env"), env.DefaultActionType(bpAPI)); err != nil {
			return err
		}
		if err := buildEnv.AddEnvDir(filepath.Join(path, "env.build"), env.DefaultActionType(bpAPI)); err != nil {
			return err
		}
	}
	return nil
}

func (d *BpDescriptor) readOutputFilesBp(bpLayersDir, bpPlanPath string, bpPlanIn Plan, bpLayers map[string]LayerMetadataFile, logger log.Logger) (BuildResult, error) {
	br := BuildResult{}
	bpFromBpInfo := GroupElement{ID: d.Buildpack.ID, Version: d.Buildpack.Version}

	// setup launch.toml
	var launchTOML LaunchTOML
	launchPath := filepath.Join(bpLayersDir, "launch.toml")

	bomValidator := NewBOMValidator(d.WithAPI, bpLayersDir, logger)

	var err error
	if api.MustParse(d.WithAPI).LessThan("0.5") {
		// read buildpack plan
		var bpPlanOut Plan
		if _, err := toml.DecodeFile(bpPlanPath, &bpPlanOut); err != nil {
			return BuildResult{}, err
		}

		// set BOM and MetRequires
		br.LaunchBOM, err = bomValidator.ValidateBOM(bpFromBpInfo, bpPlanOut.toBOM())
		if err != nil {
			return BuildResult{}, err
		}
		br.MetRequires = names(bpPlanOut.Entries)

		// set BOM files
		br.BOMFiles, err = d.processSBOMFiles(bpLayersDir, bpFromBpInfo, bpLayers, logger)
		if err != nil {
			return BuildResult{}, err
		}

		// read launch.toml, return if not exists
		if _, err := toml.DecodeFile(launchPath, &launchTOML); os.IsNotExist(err) {
			return br, nil
		} else if err != nil {
			return BuildResult{}, err
		}
	} else {
		// read build.toml
		var buildTOML BuildTOML
		buildPath := filepath.Join(bpLayersDir, "build.toml")
		if _, err := toml.DecodeFile(buildPath, &buildTOML); err != nil && !os.IsNotExist(err) {
			return BuildResult{}, err
		}
		if _, err := bomValidator.ValidateBOM(bpFromBpInfo, buildTOML.BOM); err != nil {
			return BuildResult{}, err
		}
		br.BuildBOM, err = bomValidator.ValidateBOM(bpFromBpInfo, buildTOML.BOM)
		if err != nil {
			return BuildResult{}, err
		}

		// set MetRequires
		if err := validateUnmet(buildTOML.Unmet, bpPlanIn); err != nil {
			return BuildResult{}, err
		}
		br.MetRequires = names(bpPlanIn.filter(buildTOML.Unmet).Entries)

		// set BOM files
		br.BOMFiles, err = d.processSBOMFiles(bpLayersDir, bpFromBpInfo, bpLayers, logger)
		if err != nil {
			return BuildResult{}, err
		}

		// read launch.toml, return if not exists
		if _, err := toml.DecodeFile(launchPath, &launchTOML); os.IsNotExist(err) {
			return br, nil
		} else if err != nil {
			return BuildResult{}, err
		}

		// set BOM
		br.LaunchBOM, err = bomValidator.ValidateBOM(bpFromBpInfo, launchTOML.BOM)
		if err != nil {
			return BuildResult{}, err
		}
	}

	if err := overrideDefaultForOldBuildpacks(launchTOML.Processes, d.WithAPI, logger); err != nil {
		return BuildResult{}, err
	}

	if err := validateNoMultipleDefaults(launchTOML.Processes); err != nil {
		return BuildResult{}, err
	}

	// set data from launch.toml
	br.Labels = append([]Label{}, launchTOML.Labels...)
	for i := range launchTOML.Processes {
		launchTOML.Processes[i].BuildpackID = d.Buildpack.ID
		if api.MustParse(d.WithAPI).LessThan("0.8") {
			if launchTOML.Processes[i].WorkingDirectory != "" {
				logger.Warn(fmt.Sprintf("Warning: process working directory isn't supported in this buildpack api version. Ignoring working directory for process '%s'", launchTOML.Processes[i].Type))
				launchTOML.Processes[i].WorkingDirectory = ""
			}
		}
	}
	br.Processes = append([]launch.Process{}, launchTOML.Processes...)
	br.Slices = append([]layers.Slice{}, launchTOML.Slices...)

	return br, nil
}

func names(requires []Require) []string {
	var out []string
	for _, req := range requires {
		out = append(out, req.Name)
	}
	return out
}

func validateUnmet(unmet []Unmet, bpPlan Plan) error {
	for _, unmet := range unmet {
		if unmet.Name == "" {
			return errors.New("unmet.name is required")
		}
		found := false
		for _, req := range bpPlan.Entries {
			if unmet.Name == req.Name {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unmet.name '%s' must match a requested dependency", unmet.Name)
		}
	}
	return nil
}

func overrideDefaultForOldBuildpacks(processes []launch.Process, bpAPI string, logger log.Logger) error {
	if api.MustParse(bpAPI).AtLeast("0.6") {
		return nil
	}
	var replacedDefaults []string
	for i := range processes {
		if processes[i].Default {
			replacedDefaults = append(replacedDefaults, processes[i].Type)
		}
		processes[i].Default = false
	}
	if len(replacedDefaults) > 0 {
		logger.Warn(fmt.Sprintf("Warning: default processes aren't supported in this buildpack api version. Overriding the default value to false for the following processes: [%s]", strings.Join(replacedDefaults, ", ")))
	}
	return nil
}

func validateNoMultipleDefaults(processes []launch.Process) error {
	defaultType := ""
	for _, process := range processes {
		if process.Default && defaultType != "" {
			return fmt.Errorf("multiple default process types aren't allowed")
		}
		if process.Default {
			defaultType = process.Type
		}
	}
	return nil
}

type BuildResult struct { // TODO: make GenerateResult?
	BOMFiles    []BOMFile
	BuildBOM    []BOMEntry
	Dockerfiles []DockerfileInfo
	Labels      []Label
	LaunchBOM   []BOMEntry
	MetRequires []string
	Processes   []launch.Process
	Slices      []layers.Slice
}
