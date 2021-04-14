// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/openservicemesh/osm/pkg/smi (interfaces: MeshSpec)

// Package smi is a generated GoMock package.
package smi

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"

	identity "github.com/openservicemesh/osm/pkg/identity"
	v1alpha3 "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	v1alpha4 "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	v1alpha2 "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
)

// MockMeshSpec is a mock of MeshSpec interface
type MockMeshSpec struct {
	ctrl     *gomock.Controller
	recorder *MockMeshSpecMockRecorder
}

// MockMeshSpecMockRecorder is the mock recorder for MockMeshSpec
type MockMeshSpecMockRecorder struct {
	mock *MockMeshSpec
}

// NewMockMeshSpec creates a new mock instance
func NewMockMeshSpec(ctrl *gomock.Controller) *MockMeshSpec {
	mock := &MockMeshSpec{ctrl: ctrl}
	mock.recorder = &MockMeshSpecMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockMeshSpec) EXPECT() *MockMeshSpecMockRecorder {
	return m.recorder
}

// GetTCPRoute mocks base method
func (m *MockMeshSpec) GetTCPRoute(arg0 string) *v1alpha4.TCPRoute {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTCPRoute", arg0)
	ret0, _ := ret[0].(*v1alpha4.TCPRoute)
	return ret0
}

// GetTCPRoute indicates an expected call of GetTCPRoute
func (mr *MockMeshSpecMockRecorder) GetTCPRoute(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTCPRoute", reflect.TypeOf((*MockMeshSpec)(nil).GetTCPRoute), arg0)
}

// ListHTTPTrafficSpecs mocks base method
func (m *MockMeshSpec) ListHTTPTrafficSpecs() []*v1alpha4.HTTPRouteGroup {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListHTTPTrafficSpecs")
	ret0, _ := ret[0].([]*v1alpha4.HTTPRouteGroup)
	return ret0
}

// ListHTTPTrafficSpecs indicates an expected call of ListHTTPTrafficSpecs
func (mr *MockMeshSpecMockRecorder) ListHTTPTrafficSpecs() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListHTTPTrafficSpecs", reflect.TypeOf((*MockMeshSpec)(nil).ListHTTPTrafficSpecs))
}

// ListServiceAccounts mocks base method
func (m *MockMeshSpec) ListServiceAccounts() []identity.K8sServiceAccount {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListServiceAccounts")
	ret0, _ := ret[0].([]identity.K8sServiceAccount)
	return ret0
}

// ListServiceAccounts indicates an expected call of ListServiceAccounts
func (mr *MockMeshSpecMockRecorder) ListServiceAccounts() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListServiceAccounts", reflect.TypeOf((*MockMeshSpec)(nil).ListServiceAccounts))
}

// ListTCPTrafficSpecs mocks base method
func (m *MockMeshSpec) ListTCPTrafficSpecs() []*v1alpha4.TCPRoute {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListTCPTrafficSpecs")
	ret0, _ := ret[0].([]*v1alpha4.TCPRoute)
	return ret0
}

// ListTCPTrafficSpecs indicates an expected call of ListTCPTrafficSpecs
func (mr *MockMeshSpecMockRecorder) ListTCPTrafficSpecs() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListTCPTrafficSpecs", reflect.TypeOf((*MockMeshSpec)(nil).ListTCPTrafficSpecs))
}

// ListTrafficSplits mocks base method
func (m *MockMeshSpec) ListTrafficSplits() []*v1alpha2.TrafficSplit {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListTrafficSplits")
	ret0, _ := ret[0].([]*v1alpha2.TrafficSplit)
	return ret0
}

// ListTrafficSplits indicates an expected call of ListTrafficSplits
func (mr *MockMeshSpecMockRecorder) ListTrafficSplits() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListTrafficSplits", reflect.TypeOf((*MockMeshSpec)(nil).ListTrafficSplits))
}

// ListTrafficTargets mocks base method
func (m *MockMeshSpec) ListTrafficTargets() []*v1alpha3.TrafficTarget {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListTrafficTargets")
	ret0, _ := ret[0].([]*v1alpha3.TrafficTarget)
	return ret0
}

// ListTrafficTargets indicates an expected call of ListTrafficTargets
func (mr *MockMeshSpecMockRecorder) ListTrafficTargets() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListTrafficTargets", reflect.TypeOf((*MockMeshSpec)(nil).ListTrafficTargets))
}
