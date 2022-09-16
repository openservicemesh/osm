// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/openservicemesh/osm/pkg/compute (interfaces: Interface)

// Package compute is a generated GoMock package.
package compute

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	v1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	v1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	endpoint "github.com/openservicemesh/osm/pkg/endpoint"
	envoy "github.com/openservicemesh/osm/pkg/envoy"
	identity "github.com/openservicemesh/osm/pkg/identity"
	service "github.com/openservicemesh/osm/pkg/service"
	types "k8s.io/apimachinery/pkg/types"
	cache "k8s.io/client-go/tools/cache"
)

// MockInterface is a mock of Interface interface.
type MockInterface struct {
	ctrl     *gomock.Controller
	recorder *MockInterfaceMockRecorder
}

// MockInterfaceMockRecorder is the mock recorder for MockInterface.
type MockInterfaceMockRecorder struct {
	mock *MockInterface
}

// NewMockInterface creates a new mock instance.
func NewMockInterface(ctrl *gomock.Controller) *MockInterface {
	mock := &MockInterface{ctrl: ctrl}
	mock.recorder = &MockInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockInterface) EXPECT() *MockInterfaceMockRecorder {
	return m.recorder
}

// AddMRCEventsHandler mocks base method.
func (m *MockInterface) AddMRCEventsHandler(arg0 cache.ResourceEventHandlerFuncs) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AddMRCEventsHandler", arg0)
}

// AddMRCEventsHandler indicates an expected call of AddMRCEventsHandler.
func (mr *MockInterfaceMockRecorder) AddMRCEventsHandler(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddMRCEventsHandler", reflect.TypeOf((*MockInterface)(nil).AddMRCEventsHandler), arg0)
}

// GetHostnamesForService mocks base method.
func (m *MockInterface) GetHostnamesForService(arg0 service.MeshService, arg1 bool) []string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHostnamesForService", arg0, arg1)
	ret0, _ := ret[0].([]string)
	return ret0
}

// GetHostnamesForService indicates an expected call of GetHostnamesForService.
func (mr *MockInterfaceMockRecorder) GetHostnamesForService(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHostnamesForService", reflect.TypeOf((*MockInterface)(nil).GetHostnamesForService), arg0, arg1)
}

// GetIngressBackendPolicyForService mocks base method.
func (m *MockInterface) GetIngressBackendPolicyForService(arg0 service.MeshService) *v1alpha1.IngressBackend {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetIngressBackendPolicyForService", arg0)
	ret0, _ := ret[0].(*v1alpha1.IngressBackend)
	return ret0
}

// GetIngressBackendPolicyForService indicates an expected call of GetIngressBackendPolicyForService.
func (mr *MockInterfaceMockRecorder) GetIngressBackendPolicyForService(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetIngressBackendPolicyForService", reflect.TypeOf((*MockInterface)(nil).GetIngressBackendPolicyForService), arg0)
}

// GetMeshConfig mocks base method.
func (m *MockInterface) GetMeshConfig() v1alpha2.MeshConfig {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMeshConfig")
	ret0, _ := ret[0].(v1alpha2.MeshConfig)
	return ret0
}

// GetMeshConfig indicates an expected call of GetMeshConfig.
func (mr *MockInterfaceMockRecorder) GetMeshConfig() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMeshConfig", reflect.TypeOf((*MockInterface)(nil).GetMeshConfig))
}

// GetMeshRootCertificate mocks base method.
func (m *MockInterface) GetMeshRootCertificate(arg0 string) *v1alpha2.MeshRootCertificate {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMeshRootCertificate", arg0)
	ret0, _ := ret[0].(*v1alpha2.MeshRootCertificate)
	return ret0
}

// GetMeshRootCertificate indicates an expected call of GetMeshRootCertificate.
func (mr *MockInterfaceMockRecorder) GetMeshRootCertificate(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMeshRootCertificate", reflect.TypeOf((*MockInterface)(nil).GetMeshRootCertificate), arg0)
}

// GetMeshService mocks base method.
func (m *MockInterface) GetMeshService(arg0, arg1 string, arg2 uint16) (service.MeshService, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMeshService", arg0, arg1, arg2)
	ret0, _ := ret[0].(service.MeshService)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMeshService indicates an expected call of GetMeshService.
func (mr *MockInterfaceMockRecorder) GetMeshService(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMeshService", reflect.TypeOf((*MockInterface)(nil).GetMeshService), arg0, arg1, arg2)
}

// GetOSMNamespace mocks base method.
func (m *MockInterface) GetOSMNamespace() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOSMNamespace")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetOSMNamespace indicates an expected call of GetOSMNamespace.
func (mr *MockInterfaceMockRecorder) GetOSMNamespace() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOSMNamespace", reflect.TypeOf((*MockInterface)(nil).GetOSMNamespace))
}

// GetProxyStatsHeaders mocks base method.
func (m *MockInterface) GetProxyStatsHeaders(arg0 *envoy.Proxy) (map[string]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetProxyStatsHeaders", arg0)
	ret0, _ := ret[0].(map[string]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetProxyStatsHeaders indicates an expected call of GetProxyStatsHeaders.
func (mr *MockInterfaceMockRecorder) GetProxyStatsHeaders(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetProxyStatsHeaders", reflect.TypeOf((*MockInterface)(nil).GetProxyStatsHeaders), arg0)
}

// GetResolvableEndpointsForService mocks base method.
func (m *MockInterface) GetResolvableEndpointsForService(arg0 service.MeshService) []endpoint.Endpoint {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetResolvableEndpointsForService", arg0)
	ret0, _ := ret[0].([]endpoint.Endpoint)
	return ret0
}

// GetResolvableEndpointsForService indicates an expected call of GetResolvableEndpointsForService.
func (mr *MockInterfaceMockRecorder) GetResolvableEndpointsForService(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetResolvableEndpointsForService", reflect.TypeOf((*MockInterface)(nil).GetResolvableEndpointsForService), arg0)
}

// GetServicesForServiceIdentity mocks base method.
func (m *MockInterface) GetServicesForServiceIdentity(arg0 identity.ServiceIdentity) []service.MeshService {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetServicesForServiceIdentity", arg0)
	ret0, _ := ret[0].([]service.MeshService)
	return ret0
}

// GetServicesForServiceIdentity indicates an expected call of GetServicesForServiceIdentity.
func (mr *MockInterfaceMockRecorder) GetServicesForServiceIdentity(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetServicesForServiceIdentity", reflect.TypeOf((*MockInterface)(nil).GetServicesForServiceIdentity), arg0)
}

// GetUpstreamTrafficSetting mocks base method.
func (m *MockInterface) GetUpstreamTrafficSetting(arg0 *types.NamespacedName) *v1alpha1.UpstreamTrafficSetting {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUpstreamTrafficSetting", arg0)
	ret0, _ := ret[0].(*v1alpha1.UpstreamTrafficSetting)
	return ret0
}

// GetUpstreamTrafficSetting indicates an expected call of GetUpstreamTrafficSetting.
func (mr *MockInterfaceMockRecorder) GetUpstreamTrafficSetting(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUpstreamTrafficSetting", reflect.TypeOf((*MockInterface)(nil).GetUpstreamTrafficSetting), arg0)
}

// GetUpstreamTrafficSettingByHost mocks base method.
func (m *MockInterface) GetUpstreamTrafficSettingByHost(arg0 string) *v1alpha1.UpstreamTrafficSetting {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUpstreamTrafficSettingByHost", arg0)
	ret0, _ := ret[0].(*v1alpha1.UpstreamTrafficSetting)
	return ret0
}

// GetUpstreamTrafficSettingByHost indicates an expected call of GetUpstreamTrafficSettingByHost.
func (mr *MockInterfaceMockRecorder) GetUpstreamTrafficSettingByHost(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUpstreamTrafficSettingByHost", reflect.TypeOf((*MockInterface)(nil).GetUpstreamTrafficSettingByHost), arg0)
}

// GetUpstreamTrafficSettingByNamespace mocks base method.
func (m *MockInterface) GetUpstreamTrafficSettingByNamespace(arg0 *types.NamespacedName) *v1alpha1.UpstreamTrafficSetting {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUpstreamTrafficSettingByNamespace", arg0)
	ret0, _ := ret[0].(*v1alpha1.UpstreamTrafficSetting)
	return ret0
}

// GetUpstreamTrafficSettingByNamespace indicates an expected call of GetUpstreamTrafficSettingByNamespace.
func (mr *MockInterfaceMockRecorder) GetUpstreamTrafficSettingByNamespace(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUpstreamTrafficSettingByNamespace", reflect.TypeOf((*MockInterface)(nil).GetUpstreamTrafficSettingByNamespace), arg0)
}

// GetUpstreamTrafficSettingByService mocks base method.
func (m *MockInterface) GetUpstreamTrafficSettingByService(arg0 *service.MeshService) *v1alpha1.UpstreamTrafficSetting {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUpstreamTrafficSettingByService", arg0)
	ret0, _ := ret[0].(*v1alpha1.UpstreamTrafficSetting)
	return ret0
}

// GetUpstreamTrafficSettingByService indicates an expected call of GetUpstreamTrafficSettingByService.
func (mr *MockInterfaceMockRecorder) GetUpstreamTrafficSettingByService(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUpstreamTrafficSettingByService", reflect.TypeOf((*MockInterface)(nil).GetUpstreamTrafficSettingByService), arg0)
}

// IsMetricsEnabled mocks base method.
func (m *MockInterface) IsMetricsEnabled(arg0 *envoy.Proxy) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsMetricsEnabled", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// IsMetricsEnabled indicates an expected call of IsMetricsEnabled.
func (mr *MockInterfaceMockRecorder) IsMetricsEnabled(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsMetricsEnabled", reflect.TypeOf((*MockInterface)(nil).IsMetricsEnabled), arg0)
}

// ListEgressPolicies mocks base method.
func (m *MockInterface) ListEgressPolicies() []*v1alpha1.Egress {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListEgressPolicies")
	ret0, _ := ret[0].([]*v1alpha1.Egress)
	return ret0
}

// ListEgressPolicies indicates an expected call of ListEgressPolicies.
func (mr *MockInterfaceMockRecorder) ListEgressPolicies() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListEgressPolicies", reflect.TypeOf((*MockInterface)(nil).ListEgressPolicies))
}

// ListEgressPoliciesForServiceAccount mocks base method.
func (m *MockInterface) ListEgressPoliciesForServiceAccount(arg0 identity.K8sServiceAccount) []*v1alpha1.Egress {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListEgressPoliciesForServiceAccount", arg0)
	ret0, _ := ret[0].([]*v1alpha1.Egress)
	return ret0
}

// ListEgressPoliciesForServiceAccount indicates an expected call of ListEgressPoliciesForServiceAccount.
func (mr *MockInterfaceMockRecorder) ListEgressPoliciesForServiceAccount(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListEgressPoliciesForServiceAccount", reflect.TypeOf((*MockInterface)(nil).ListEgressPoliciesForServiceAccount), arg0)
}

// ListEndpointsForIdentity mocks base method.
func (m *MockInterface) ListEndpointsForIdentity(arg0 identity.ServiceIdentity) []endpoint.Endpoint {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListEndpointsForIdentity", arg0)
	ret0, _ := ret[0].([]endpoint.Endpoint)
	return ret0
}

// ListEndpointsForIdentity indicates an expected call of ListEndpointsForIdentity.
func (mr *MockInterfaceMockRecorder) ListEndpointsForIdentity(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListEndpointsForIdentity", reflect.TypeOf((*MockInterface)(nil).ListEndpointsForIdentity), arg0)
}

// ListEndpointsForService mocks base method.
func (m *MockInterface) ListEndpointsForService(arg0 service.MeshService) []endpoint.Endpoint {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListEndpointsForService", arg0)
	ret0, _ := ret[0].([]endpoint.Endpoint)
	return ret0
}

// ListEndpointsForService indicates an expected call of ListEndpointsForService.
func (mr *MockInterfaceMockRecorder) ListEndpointsForService(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListEndpointsForService", reflect.TypeOf((*MockInterface)(nil).ListEndpointsForService), arg0)
}

// ListIngressBackendPolicies mocks base method.
func (m *MockInterface) ListIngressBackendPolicies() []*v1alpha1.IngressBackend {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListIngressBackendPolicies")
	ret0, _ := ret[0].([]*v1alpha1.IngressBackend)
	return ret0
}

// ListIngressBackendPolicies indicates an expected call of ListIngressBackendPolicies.
func (mr *MockInterfaceMockRecorder) ListIngressBackendPolicies() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListIngressBackendPolicies", reflect.TypeOf((*MockInterface)(nil).ListIngressBackendPolicies))
}

// ListMeshRootCertificates mocks base method.
func (m *MockInterface) ListMeshRootCertificates() ([]*v1alpha2.MeshRootCertificate, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListMeshRootCertificates")
	ret0, _ := ret[0].([]*v1alpha2.MeshRootCertificate)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListMeshRootCertificates indicates an expected call of ListMeshRootCertificates.
func (mr *MockInterfaceMockRecorder) ListMeshRootCertificates() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListMeshRootCertificates", reflect.TypeOf((*MockInterface)(nil).ListMeshRootCertificates))
}

// ListNamespaces mocks base method.
func (m *MockInterface) ListNamespaces() ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListNamespaces")
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListNamespaces indicates an expected call of ListNamespaces.
func (mr *MockInterfaceMockRecorder) ListNamespaces() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListNamespaces", reflect.TypeOf((*MockInterface)(nil).ListNamespaces))
}

// ListRetryPolicies mocks base method.
func (m *MockInterface) ListRetryPolicies() []*v1alpha1.Retry {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRetryPolicies")
	ret0, _ := ret[0].([]*v1alpha1.Retry)
	return ret0
}

// ListRetryPolicies indicates an expected call of ListRetryPolicies.
func (mr *MockInterfaceMockRecorder) ListRetryPolicies() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRetryPolicies", reflect.TypeOf((*MockInterface)(nil).ListRetryPolicies))
}

// ListRetryPoliciesForServiceAccount mocks base method.
func (m *MockInterface) ListRetryPoliciesForServiceAccount(arg0 identity.K8sServiceAccount) []*v1alpha1.Retry {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRetryPoliciesForServiceAccount", arg0)
	ret0, _ := ret[0].([]*v1alpha1.Retry)
	return ret0
}

// ListRetryPoliciesForServiceAccount indicates an expected call of ListRetryPoliciesForServiceAccount.
func (mr *MockInterfaceMockRecorder) ListRetryPoliciesForServiceAccount(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRetryPoliciesForServiceAccount", reflect.TypeOf((*MockInterface)(nil).ListRetryPoliciesForServiceAccount), arg0)
}

// ListServiceIdentitiesForService mocks base method.
func (m *MockInterface) ListServiceIdentitiesForService(arg0, arg1 string) ([]identity.ServiceIdentity, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListServiceIdentitiesForService", arg0, arg1)
	ret0, _ := ret[0].([]identity.ServiceIdentity)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListServiceIdentitiesForService indicates an expected call of ListServiceIdentitiesForService.
func (mr *MockInterfaceMockRecorder) ListServiceIdentitiesForService(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListServiceIdentitiesForService", reflect.TypeOf((*MockInterface)(nil).ListServiceIdentitiesForService), arg0, arg1)
}

// ListServices mocks base method.
func (m *MockInterface) ListServices() []service.MeshService {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListServices")
	ret0, _ := ret[0].([]service.MeshService)
	return ret0
}

// ListServices indicates an expected call of ListServices.
func (mr *MockInterfaceMockRecorder) ListServices() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListServices", reflect.TypeOf((*MockInterface)(nil).ListServices))
}

// ListServicesForProxy mocks base method.
func (m *MockInterface) ListServicesForProxy(arg0 *envoy.Proxy) ([]service.MeshService, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListServicesForProxy", arg0)
	ret0, _ := ret[0].([]service.MeshService)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListServicesForProxy indicates an expected call of ListServicesForProxy.
func (mr *MockInterfaceMockRecorder) ListServicesForProxy(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListServicesForProxy", reflect.TypeOf((*MockInterface)(nil).ListServicesForProxy), arg0)
}

// ListUpstreamTrafficSettings mocks base method.
func (m *MockInterface) ListUpstreamTrafficSettings() []*v1alpha1.UpstreamTrafficSetting {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListUpstreamTrafficSettings")
	ret0, _ := ret[0].([]*v1alpha1.UpstreamTrafficSetting)
	return ret0
}

// ListUpstreamTrafficSettings indicates an expected call of ListUpstreamTrafficSettings.
func (mr *MockInterfaceMockRecorder) ListUpstreamTrafficSettings() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListUpstreamTrafficSettings", reflect.TypeOf((*MockInterface)(nil).ListUpstreamTrafficSettings))
}

// UpdateIngressBackendStatus mocks base method.
func (m *MockInterface) UpdateIngressBackendStatus(arg0 *v1alpha1.IngressBackend) (*v1alpha1.IngressBackend, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateIngressBackendStatus", arg0)
	ret0, _ := ret[0].(*v1alpha1.IngressBackend)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateIngressBackendStatus indicates an expected call of UpdateIngressBackendStatus.
func (mr *MockInterfaceMockRecorder) UpdateIngressBackendStatus(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateIngressBackendStatus", reflect.TypeOf((*MockInterface)(nil).UpdateIngressBackendStatus), arg0)
}

// UpdateMeshRootCertificate mocks base method.
func (m *MockInterface) UpdateMeshRootCertificate(arg0 *v1alpha2.MeshRootCertificate) (*v1alpha2.MeshRootCertificate, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateMeshRootCertificate", arg0)
	ret0, _ := ret[0].(*v1alpha2.MeshRootCertificate)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateMeshRootCertificate indicates an expected call of UpdateMeshRootCertificate.
func (mr *MockInterfaceMockRecorder) UpdateMeshRootCertificate(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateMeshRootCertificate", reflect.TypeOf((*MockInterface)(nil).UpdateMeshRootCertificate), arg0)
}

// UpdateMeshRootCertificateStatus mocks base method.
func (m *MockInterface) UpdateMeshRootCertificateStatus(arg0 *v1alpha2.MeshRootCertificate) (*v1alpha2.MeshRootCertificate, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateMeshRootCertificateStatus", arg0)
	ret0, _ := ret[0].(*v1alpha2.MeshRootCertificate)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateMeshRootCertificateStatus indicates an expected call of UpdateMeshRootCertificateStatus.
func (mr *MockInterfaceMockRecorder) UpdateMeshRootCertificateStatus(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateMeshRootCertificateStatus", reflect.TypeOf((*MockInterface)(nil).UpdateMeshRootCertificateStatus), arg0)
}

// UpdateUpstreamTrafficSettingStatus mocks base method.
func (m *MockInterface) UpdateUpstreamTrafficSettingStatus(arg0 *v1alpha1.UpstreamTrafficSetting) (*v1alpha1.UpstreamTrafficSetting, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateUpstreamTrafficSettingStatus", arg0)
	ret0, _ := ret[0].(*v1alpha1.UpstreamTrafficSetting)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateUpstreamTrafficSettingStatus indicates an expected call of UpdateUpstreamTrafficSettingStatus.
func (mr *MockInterfaceMockRecorder) UpdateUpstreamTrafficSettingStatus(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateUpstreamTrafficSettingStatus", reflect.TypeOf((*MockInterface)(nil).UpdateUpstreamTrafficSettingStatus), arg0)
}

// VerifyProxy mocks base method.
func (m *MockInterface) VerifyProxy(arg0 *envoy.Proxy) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "VerifyProxy", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// VerifyProxy indicates an expected call of VerifyProxy.
func (mr *MockInterfaceMockRecorder) VerifyProxy(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "VerifyProxy", reflect.TypeOf((*MockInterface)(nil).VerifyProxy), arg0)
}
