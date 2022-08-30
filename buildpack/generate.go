package buildpack

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/buildpacks/lifecycle/launch"
	"github.com/buildpacks/lifecycle/log"
)

type GenerateInputs struct {
	AppDir      string
	OutputDir   string
	PlatformDir string
	Env         BuildEnv
	Out, Err    io.Writer
	Plan        Plan
}

type GenerateOutputs struct {
	Dockerfiles []DockerfileInfo
	MetRequires []string
}

//go:generate mockgen -package testmock -destination ../testmock/generate_executor.go github.com/buildpacks/lifecycle/buildpack GenerateExecutor
type GenerateExecutor interface {
	Generate(d ExtDescriptor, inputs GenerateInputs, logger log.Logger) (GenerateOutputs, error)
}

type DefaultGenerateExecutor struct{}

func (e *DefaultGenerateExecutor) Generate(d ExtDescriptor, inputs GenerateInputs, logger log.Logger) (GenerateOutputs, error) { // TODO: fix other pointer arguments (Build, Detect)
	logger.Debug("Creating plan directory")
	planDir, err := ioutil.TempDir("", launch.EscapeID(d.Extension.ID)+"-")
	if err != nil {
		return GenerateOutputs{}, err
	}
	defer os.RemoveAll(planDir)

	logger.Debug("Preparing paths")
	moduleOutputDir, planPath, err := prepareInputPaths(d.Extension.ID, inputs.Plan, inputs.OutputDir, planDir)
	if err != nil {
		return GenerateOutputs{}, err
	}

	logger.Debug("Running generate command")
	if _, err = os.Stat(filepath.Join(d.WithRootDir, "bin", "generate")); err != nil {
		if os.IsNotExist(err) {
			// treat extension root directory as pre-populated output directory
			return readOutputFilesExt(d, filepath.Join(d.WithRootDir, "generate"), inputs.Plan)
		}
		return GenerateOutputs{}, err
	}
	if err = runGenerateCmd(d, moduleOutputDir, planPath, inputs); err != nil {
		return GenerateOutputs{}, err
	}

	logger.Debug("Reading output files")
	return readOutputFilesExt(d, moduleOutputDir, inputs.Plan)
}

func runGenerateCmd(d ExtDescriptor, moduleOutputDir, planPath string, inputs GenerateInputs) error {
	cmd := exec.Command(
		filepath.Join(d.WithRootDir, "bin", "generate"),
		moduleOutputDir,
		inputs.PlatformDir,
		planPath,
	) // #nosec G204
	cmd.Dir = inputs.AppDir
	cmd.Stdout = inputs.Out
	cmd.Stderr = inputs.Err

	var err error
	if d.Extension.ClearEnv {
		cmd.Env = inputs.Env.List()
	} else {
		cmd.Env, err = inputs.Env.WithPlatform(inputs.PlatformDir)
		if err != nil {
			return err
		}
	}
	cmd.Env = append(cmd.Env,
		EnvBpPlanPath+"="+planPath,
		EnvBuildpackDir+"="+d.WithRootDir, // TODO: should be extension dir?
		EnvOutputDir+"="+moduleOutputDir,
		EnvPlatformDir+"="+inputs.PlatformDir,
	)

	if err := cmd.Run(); err != nil {
		return NewError(err, ErrTypeBuildpack)
	}
	return nil
}

func readOutputFilesExt(d ExtDescriptor, extOutputDir string, extPlanIn Plan) (GenerateOutputs, error) {
	br := GenerateOutputs{}
	var err error

	// set MetRequires
	br.MetRequires = names(extPlanIn.Entries)

	// set Dockerfiles
	runDockerfile := filepath.Join(extOutputDir, "run.Dockerfile")
	if _, err = os.Stat(runDockerfile); err != nil {
		if os.IsNotExist(err) {
			return br, nil
		}
		return GenerateOutputs{}, err
	}
	br.Dockerfiles = []DockerfileInfo{{ExtensionID: d.Extension.ID, Kind: DockerfileKindRun, Path: runDockerfile}}
	return br, nil
}
