// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/buildpacks/lifecycle (interfaces: Buildpack)

// Package testmock is a generated GoMock package.
package testmock

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"

	buildpack "github.com/buildpacks/lifecycle/buildpack"
)

// MockBuildpack is a mock of Buildpack interface.
type MockBuildpack struct {
	ctrl     *gomock.Controller
	recorder *MockBuildpackMockRecorder
}

// MockBuildpackMockRecorder is the mock recorder for MockBuildpack.
type MockBuildpackMockRecorder struct {
	mock *MockBuildpack
}

// NewMockBuildpack creates a new mock instance.
func NewMockBuildpack(ctrl *gomock.Controller) *MockBuildpack {
	mock := &MockBuildpack{ctrl: ctrl}
	mock.recorder = &MockBuildpackMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBuildpack) EXPECT() *MockBuildpackMockRecorder {
	return m.recorder
}

// Build mocks base method.
func (m *MockBuildpack) Build(arg0 buildpack.Plan, arg1 buildpack.BuildConfig, arg2 buildpack.BuildEnv) (buildpack.BuildResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Build", arg0, arg1, arg2)
	ret0, _ := ret[0].(buildpack.BuildResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Build indicates an expected call of Build.
func (mr *MockBuildpackMockRecorder) Build(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Build", reflect.TypeOf((*MockBuildpack)(nil).Build), arg0, arg1, arg2)
}

// ConfigFile mocks base method.
func (m *MockBuildpack) ConfigFile() *buildpack.Descriptor {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConfigFile")
	ret0, _ := ret[0].(*buildpack.Descriptor)
	return ret0
}

// ConfigFile indicates an expected call of ConfigFile.
func (mr *MockBuildpackMockRecorder) ConfigFile() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConfigFile", reflect.TypeOf((*MockBuildpack)(nil).ConfigFile))
}

// Detect mocks base method.
func (m *MockBuildpack) Detect(arg0 *buildpack.DetectConfig, arg1 buildpack.BuildEnv) buildpack.DetectRun {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Detect", arg0, arg1)
	ret0, _ := ret[0].(buildpack.DetectRun)
	return ret0
}

// Detect indicates an expected call of Detect.
func (mr *MockBuildpackMockRecorder) Detect(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Detect", reflect.TypeOf((*MockBuildpack)(nil).Detect), arg0, arg1)
}

// SupportsAssetPackages mocks base method.
func (m *MockBuildpack) SupportsAssetPackages() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SupportsAssetPackages")
	ret0, _ := ret[0].(bool)
	return ret0
}

// SupportsAssetPackages indicates an expected call of SupportsAssetPackages.
func (mr *MockBuildpackMockRecorder) SupportsAssetPackages() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SupportsAssetPackages", reflect.TypeOf((*MockBuildpack)(nil).SupportsAssetPackages))
}
