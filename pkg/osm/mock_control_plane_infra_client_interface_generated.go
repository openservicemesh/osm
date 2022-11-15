// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/openservicemesh/osm/pkg/osm (interfaces: ControlPlaneInfraClient)

// Package osm is a generated GoMock package.
package osm

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	v1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	models "github.com/openservicemesh/osm/pkg/models"
)

// MockControlPlaneInfraClient is a mock of ControlPlaneInfraClient interface.
type MockControlPlaneInfraClient struct {
	ctrl     *gomock.Controller
	recorder *MockControlPlaneInfraClientMockRecorder
}

// MockControlPlaneInfraClientMockRecorder is the mock recorder for MockControlPlaneInfraClient.
type MockControlPlaneInfraClientMockRecorder struct {
	mock *MockControlPlaneInfraClient
}

// NewMockControlPlaneInfraClient creates a new mock instance.
func NewMockControlPlaneInfraClient(ctrl *gomock.Controller) *MockControlPlaneInfraClient {
	mock := &MockControlPlaneInfraClient{ctrl: ctrl}
	mock.recorder = &MockControlPlaneInfraClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockControlPlaneInfraClient) EXPECT() *MockControlPlaneInfraClientMockRecorder {
	return m.recorder
}

// GetMeshConfig mocks base method.
func (m *MockControlPlaneInfraClient) GetMeshConfig() v1alpha2.MeshConfig {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMeshConfig")
	ret0, _ := ret[0].(v1alpha2.MeshConfig)
	return ret0
}

// GetMeshConfig indicates an expected call of GetMeshConfig.
func (mr *MockControlPlaneInfraClientMockRecorder) GetMeshConfig() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMeshConfig", reflect.TypeOf((*MockControlPlaneInfraClient)(nil).GetMeshConfig))
}

// VerifyProxy mocks base method.
func (m *MockControlPlaneInfraClient) VerifyProxy(arg0 *models.Proxy) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "VerifyProxy", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// VerifyProxy indicates an expected call of VerifyProxy.
func (mr *MockControlPlaneInfraClientMockRecorder) VerifyProxy(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "VerifyProxy", reflect.TypeOf((*MockControlPlaneInfraClient)(nil).VerifyProxy), arg0)
}
