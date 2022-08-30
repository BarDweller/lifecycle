// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/buildpacks/lifecycle/buildpack (interfaces: GenerateExecutor)

// Package testmock is a generated GoMock package.
package testmock

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"

	buildpack "github.com/buildpacks/lifecycle/buildpack"
)

// MockGenerateExecutor is a mock of GenerateExecutor interface.
type MockGenerateExecutor struct {
	ctrl     *gomock.Controller
	recorder *MockGenerateExecutorMockRecorder
}

// MockGenerateExecutorMockRecorder is the mock recorder for MockGenerateExecutor.
type MockGenerateExecutorMockRecorder struct {
	mock *MockGenerateExecutor
}

// NewMockGenerateExecutor creates a new mock instance.
func NewMockGenerateExecutor(ctrl *gomock.Controller) *MockGenerateExecutor {
	mock := &MockGenerateExecutor{ctrl: ctrl}
	mock.recorder = &MockGenerateExecutorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockGenerateExecutor) EXPECT() *MockGenerateExecutorMockRecorder {
	return m.recorder
}

// Generate mocks base method.
func (m *MockGenerateExecutor) Generate(arg0 buildpack.ExtDescriptor, arg1 buildpack.Plan, arg2 buildpack.BuildConfig, arg3 buildpack.BuildEnv) (buildpack.BuildResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Generate", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(buildpack.BuildResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Generate indicates an expected call of Generate.
func (mr *MockGenerateExecutorMockRecorder) Generate(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Generate", reflect.TypeOf((*MockGenerateExecutor)(nil).Generate), arg0, arg1, arg2, arg3)
}
