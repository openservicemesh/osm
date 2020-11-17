// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/certificate/types.go

// Package certificate is a generated GoMock package.
package certificate

import (
	reflect "reflect"
	time "time"

	gomock "github.com/golang/mock/gomock"
	"github.com/openservicemesh/osm/pkg/announcements"
)

// MockCertificater is a mock of Certificater interface
type MockCertificater struct {
	ctrl     *gomock.Controller
	recorder *MockCertificaterMockRecorder
}

// MockCertificaterMockRecorder is the mock recorder for MockCertificater
type MockCertificaterMockRecorder struct {
	mock *MockCertificater
}

// NewMockCertificater creates a new mock instance
func NewMockCertificater(ctrl *gomock.Controller) *MockCertificater {
	mock := &MockCertificater{ctrl: ctrl}
	mock.recorder = &MockCertificaterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockCertificater) EXPECT() *MockCertificaterMockRecorder {
	return m.recorder
}

// GetCommonName mocks base method
func (m *MockCertificater) GetCommonName() CommonName {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCommonName")
	ret0, _ := ret[0].(CommonName)
	return ret0
}

// GetCommonName indicates an expected call of GetCommonName
func (mr *MockCertificaterMockRecorder) GetCommonName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCommonName", reflect.TypeOf((*MockCertificater)(nil).GetCommonName))
}

// GetCertificateChain mocks base method
func (m *MockCertificater) GetCertificateChain() []byte {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCertificateChain")
	ret0, _ := ret[0].([]byte)
	return ret0
}

// GetCertificateChain indicates an expected call of GetCertificateChain
func (mr *MockCertificaterMockRecorder) GetCertificateChain() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCertificateChain", reflect.TypeOf((*MockCertificater)(nil).GetCertificateChain))
}

// GetPrivateKey mocks base method
func (m *MockCertificater) GetPrivateKey() []byte {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPrivateKey")
	ret0, _ := ret[0].([]byte)
	return ret0
}

// GetPrivateKey indicates an expected call of GetPrivateKey
func (mr *MockCertificaterMockRecorder) GetPrivateKey() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPrivateKey", reflect.TypeOf((*MockCertificater)(nil).GetPrivateKey))
}

// GetIssuingCA mocks base method
func (m *MockCertificater) GetIssuingCA() []byte {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetIssuingCA")
	ret0, _ := ret[0].([]byte)
	return ret0
}

// GetIssuingCA indicates an expected call of GetIssuingCA
func (mr *MockCertificaterMockRecorder) GetIssuingCA() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetIssuingCA", reflect.TypeOf((*MockCertificater)(nil).GetIssuingCA))
}

// GetExpiration mocks base method
func (m *MockCertificater) GetExpiration() time.Time {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetExpiration")
	ret0, _ := ret[0].(time.Time)
	return ret0
}

// GetExpiration indicates an expected call of GetExpiration
func (mr *MockCertificaterMockRecorder) GetExpiration() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetExpiration", reflect.TypeOf((*MockCertificater)(nil).GetExpiration))
}

// MockManager is a mock of Manager interface
type MockManager struct {
	ctrl     *gomock.Controller
	recorder *MockManagerMockRecorder
}

// MockManagerMockRecorder is the mock recorder for MockManager
type MockManagerMockRecorder struct {
	mock *MockManager
}

// NewMockManager creates a new mock instance
func NewMockManager(ctrl *gomock.Controller) *MockManager {
	mock := &MockManager{ctrl: ctrl}
	mock.recorder = &MockManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockManager) EXPECT() *MockManagerMockRecorder {
	return m.recorder
}

// IssueCertificate mocks base method
func (m *MockManager) IssueCertificate(arg0 CommonName, arg1 time.Duration) (Certificater, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IssueCertificate", arg0, arg1)
	ret0, _ := ret[0].(Certificater)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// IssueCertificate indicates an expected call of IssueCertificate
func (mr *MockManagerMockRecorder) IssueCertificate(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IssueCertificate", reflect.TypeOf((*MockManager)(nil).IssueCertificate), arg0, arg1)
}

// GetCertificate mocks base method
func (m *MockManager) GetCertificate(arg0 CommonName) (Certificater, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCertificate", arg0)
	ret0, _ := ret[0].(Certificater)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCertificate indicates an expected call of GetCertificate
func (mr *MockManagerMockRecorder) GetCertificate(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCertificate", reflect.TypeOf((*MockManager)(nil).GetCertificate), arg0)
}

// RotateCertificate mocks base method
func (m *MockManager) RotateCertificate(arg0 CommonName) (Certificater, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RotateCertificate", arg0)
	ret0, _ := ret[0].(Certificater)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RotateCertificate indicates an expected call of RotateCertificate
func (mr *MockManagerMockRecorder) RotateCertificate(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RotateCertificate", reflect.TypeOf((*MockManager)(nil).RotateCertificate), arg0)
}

// GetRootCertificate mocks base method
func (m *MockManager) GetRootCertificate() (Certificater, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRootCertificate")
	ret0, _ := ret[0].(Certificater)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRootCertificate indicates an expected call of GetRootCertificate
func (mr *MockManagerMockRecorder) GetRootCertificate() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRootCertificate", reflect.TypeOf((*MockManager)(nil).GetRootCertificate))
}

// ListCertificates mocks base method
func (m *MockManager) ListCertificates() ([]Certificater, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListCertificates")
	ret0, _ := ret[0].([]Certificater)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListCertificates indicates an expected call of ListCertificates
func (mr *MockManagerMockRecorder) ListCertificates() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListCertificates", reflect.TypeOf((*MockManager)(nil).ListCertificates))
}

// ReleaseCertificate mocks base method
func (m *MockManager) ReleaseCertificate(arg0 CommonName) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "ReleaseCertificate", arg0)
}

// ReleaseCertificate indicates an expected call of ReleaseCertificate
func (mr *MockManagerMockRecorder) ReleaseCertificate(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReleaseCertificate", reflect.TypeOf((*MockManager)(nil).ReleaseCertificate), arg0)
}

// GetAnnouncementsChannel mocks base method
func (m *MockManager) GetAnnouncementsChannel() <-chan announcements.Announcement {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAnnouncementsChannel")
	ret0, _ := ret[0].(<-chan announcements.Announcement)
	return ret0
}

// GetAnnouncementsChannel indicates an expected call of GetAnnouncementsChannel
func (mr *MockManagerMockRecorder) GetAnnouncementsChannel() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAnnouncementsChannel", reflect.TypeOf((*MockManager)(nil).GetAnnouncementsChannel))
}
