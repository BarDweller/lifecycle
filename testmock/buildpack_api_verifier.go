// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/buildpacks/lifecycle (interfaces: BuildpackAPIVerifier)

// Package testmock is a generated GoMock package.
package testmock

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"

	log "github.com/buildpacks/lifecycle/log"
)

// MockBuildpackAPIVerifier is a mock of BuildpackAPIVerifier interface.
type MockBuildpackAPIVerifier struct {
	ctrl     *gomock.Controller
	recorder *MockBuildpackAPIVerifierMockRecorder
}

// MockBuildpackAPIVerifierMockRecorder is the mock recorder for MockBuildpackAPIVerifier.
type MockBuildpackAPIVerifierMockRecorder struct {
	mock *MockBuildpackAPIVerifier
}

// NewMockBuildpackAPIVerifier creates a new mock instance.
func NewMockBuildpackAPIVerifier(ctrl *gomock.Controller) *MockBuildpackAPIVerifier {
	mock := &MockBuildpackAPIVerifier{ctrl: ctrl}
	mock.recorder = &MockBuildpackAPIVerifierMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBuildpackAPIVerifier) EXPECT() *MockBuildpackAPIVerifierMockRecorder {
	return m.recorder
}

// VerifyBuildpackAPI mocks base method.
func (m *MockBuildpackAPIVerifier) VerifyBuildpackAPI(arg0, arg1, arg2 string, arg3 log.Logger) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "VerifyBuildpackAPI", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// VerifyBuildpackAPI indicates an expected call of VerifyBuildpackAPI.
func (mr *MockBuildpackAPIVerifierMockRecorder) VerifyBuildpackAPI(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "VerifyBuildpackAPI", reflect.TypeOf((*MockBuildpackAPIVerifier)(nil).VerifyBuildpackAPI), arg0, arg1, arg2, arg3)
}
