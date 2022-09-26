// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/openservicemesh/osm/pkg/catalog (interfaces: MeshCataloger)

// Package catalog is a generated GoMock package.
package catalog

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	v1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	v1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	endpoint "github.com/openservicemesh/osm/pkg/endpoint"
	identity "github.com/openservicemesh/osm/pkg/identity"
	models "github.com/openservicemesh/osm/pkg/models"
	service "github.com/openservicemesh/osm/pkg/service"
	trafficpolicy "github.com/openservicemesh/osm/pkg/trafficpolicy"
	v1alpha3 "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	v1alpha4 "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	v1alpha20 "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	types "k8s.io/apimachinery/pkg/types"
)

// MockMeshCataloger is a mock of MeshCataloger interface.
type MockMeshCataloger struct {
	ctrl     *gomock.Controller
	recorder *MockMeshCatalogerMockRecorder
}

// MockMeshCatalogerMockRecorder is the mock recorder for MockMeshCataloger.
type MockMeshCatalogerMockRecorder struct {
	mock *MockMeshCataloger
}

// NewMockMeshCataloger creates a new mock instance.
func NewMockMeshCataloger(ctrl *gomock.Controller) *MockMeshCataloger {
	mock := &MockMeshCataloger{ctrl: ctrl}
	mock.recorder = &MockMeshCatalogerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMeshCataloger) EXPECT() *MockMeshCatalogerMockRecorder {
	return m.recorder
}

// GetEgressTrafficPolicy mocks base method.
func (m *MockMeshCataloger) GetEgressTrafficPolicy(arg0 identity.ServiceIdentity) (*trafficpolicy.EgressTrafficPolicy, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEgressTrafficPolicy", arg0)
	ret0, _ := ret[0].(*trafficpolicy.EgressTrafficPolicy)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetEgressTrafficPolicy indicates an expected call of GetEgressTrafficPolicy.
func (mr *MockMeshCatalogerMockRecorder) GetEgressTrafficPolicy(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEgressTrafficPolicy", reflect.TypeOf((*MockMeshCataloger)(nil).GetEgressTrafficPolicy), arg0)
}

// GetHTTPRouteGroup mocks base method.
func (m *MockMeshCataloger) GetHTTPRouteGroup(arg0 string) *v1alpha4.HTTPRouteGroup {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHTTPRouteGroup", arg0)
	ret0, _ := ret[0].(*v1alpha4.HTTPRouteGroup)
	return ret0
}

// GetHTTPRouteGroup indicates an expected call of GetHTTPRouteGroup.
func (mr *MockMeshCatalogerMockRecorder) GetHTTPRouteGroup(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHTTPRouteGroup", reflect.TypeOf((*MockMeshCataloger)(nil).GetHTTPRouteGroup), arg0)
}

// GetHostnamesForService mocks base method.
func (m *MockMeshCataloger) GetHostnamesForService(arg0 service.MeshService, arg1 bool) []string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHostnamesForService", arg0, arg1)
	ret0, _ := ret[0].([]string)
	return ret0
}

// GetHostnamesForService indicates an expected call of GetHostnamesForService.
func (mr *MockMeshCatalogerMockRecorder) GetHostnamesForService(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHostnamesForService", reflect.TypeOf((*MockMeshCataloger)(nil).GetHostnamesForService), arg0, arg1)
}

// GetInboundMeshTrafficPolicy mocks base method.
func (m *MockMeshCataloger) GetInboundMeshTrafficPolicy(arg0 identity.ServiceIdentity, arg1 []service.MeshService) *trafficpolicy.InboundMeshTrafficPolicy {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetInboundMeshTrafficPolicy", arg0, arg1)
	ret0, _ := ret[0].(*trafficpolicy.InboundMeshTrafficPolicy)
	return ret0
}

// GetInboundMeshTrafficPolicy indicates an expected call of GetInboundMeshTrafficPolicy.
func (mr *MockMeshCatalogerMockRecorder) GetInboundMeshTrafficPolicy(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetInboundMeshTrafficPolicy", reflect.TypeOf((*MockMeshCataloger)(nil).GetInboundMeshTrafficPolicy), arg0, arg1)
}

// GetIngressBackendPolicyForService mocks base method.
func (m *MockMeshCataloger) GetIngressBackendPolicyForService(arg0 service.MeshService) *v1alpha1.IngressBackend {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetIngressBackendPolicyForService", arg0)
	ret0, _ := ret[0].(*v1alpha1.IngressBackend)
	return ret0
}

// GetIngressBackendPolicyForService indicates an expected call of GetIngressBackendPolicyForService.
func (mr *MockMeshCatalogerMockRecorder) GetIngressBackendPolicyForService(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetIngressBackendPolicyForService", reflect.TypeOf((*MockMeshCataloger)(nil).GetIngressBackendPolicyForService), arg0)
}

// GetIngressTrafficPolicies mocks base method.
func (m *MockMeshCataloger) GetIngressTrafficPolicies(arg0 []service.MeshService) []*trafficpolicy.IngressTrafficPolicy {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetIngressTrafficPolicies", arg0)
	ret0, _ := ret[0].([]*trafficpolicy.IngressTrafficPolicy)
	return ret0
}

// GetIngressTrafficPolicies indicates an expected call of GetIngressTrafficPolicies.
func (mr *MockMeshCatalogerMockRecorder) GetIngressTrafficPolicies(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetIngressTrafficPolicies", reflect.TypeOf((*MockMeshCataloger)(nil).GetIngressTrafficPolicies), arg0)
}

// GetIngressTrafficPolicy mocks base method.
func (m *MockMeshCataloger) GetIngressTrafficPolicy(arg0 service.MeshService) (*trafficpolicy.IngressTrafficPolicy, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetIngressTrafficPolicy", arg0)
	ret0, _ := ret[0].(*trafficpolicy.IngressTrafficPolicy)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetIngressTrafficPolicy indicates an expected call of GetIngressTrafficPolicy.
func (mr *MockMeshCatalogerMockRecorder) GetIngressTrafficPolicy(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetIngressTrafficPolicy", reflect.TypeOf((*MockMeshCataloger)(nil).GetIngressTrafficPolicy), arg0)
}

// GetMeshConfig mocks base method.
func (m *MockMeshCataloger) GetMeshConfig() v1alpha2.MeshConfig {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMeshConfig")
	ret0, _ := ret[0].(v1alpha2.MeshConfig)
	return ret0
}

// GetMeshConfig indicates an expected call of GetMeshConfig.
func (mr *MockMeshCatalogerMockRecorder) GetMeshConfig() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMeshConfig", reflect.TypeOf((*MockMeshCataloger)(nil).GetMeshConfig))
}

// GetMeshRootCertificate mocks base method.
func (m *MockMeshCataloger) GetMeshRootCertificate(arg0 string) *v1alpha2.MeshRootCertificate {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMeshRootCertificate", arg0)
	ret0, _ := ret[0].(*v1alpha2.MeshRootCertificate)
	return ret0
}

// GetMeshRootCertificate indicates an expected call of GetMeshRootCertificate.
func (mr *MockMeshCatalogerMockRecorder) GetMeshRootCertificate(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMeshRootCertificate", reflect.TypeOf((*MockMeshCataloger)(nil).GetMeshRootCertificate), arg0)
}

// GetMeshService mocks base method.
func (m *MockMeshCataloger) GetMeshService(arg0, arg1 string, arg2 uint16) (service.MeshService, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMeshService", arg0, arg1, arg2)
	ret0, _ := ret[0].(service.MeshService)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMeshService indicates an expected call of GetMeshService.
func (mr *MockMeshCatalogerMockRecorder) GetMeshService(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMeshService", reflect.TypeOf((*MockMeshCataloger)(nil).GetMeshService), arg0, arg1, arg2)
}

// GetOSMNamespace mocks base method.
func (m *MockMeshCataloger) GetOSMNamespace() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOSMNamespace")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetOSMNamespace indicates an expected call of GetOSMNamespace.
func (mr *MockMeshCatalogerMockRecorder) GetOSMNamespace() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOSMNamespace", reflect.TypeOf((*MockMeshCataloger)(nil).GetOSMNamespace))
}

// GetOutboundMeshTrafficPolicy mocks base method.
func (m *MockMeshCataloger) GetOutboundMeshTrafficPolicy(arg0 identity.ServiceIdentity) *trafficpolicy.OutboundMeshTrafficPolicy {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOutboundMeshTrafficPolicy", arg0)
	ret0, _ := ret[0].(*trafficpolicy.OutboundMeshTrafficPolicy)
	return ret0
}

// GetOutboundMeshTrafficPolicy indicates an expected call of GetOutboundMeshTrafficPolicy.
func (mr *MockMeshCatalogerMockRecorder) GetOutboundMeshTrafficPolicy(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOutboundMeshTrafficPolicy", reflect.TypeOf((*MockMeshCataloger)(nil).GetOutboundMeshTrafficPolicy), arg0)
}

// GetProxyStatsHeaders mocks base method.
func (m *MockMeshCataloger) GetProxyStatsHeaders(arg0 *models.Proxy) (map[string]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetProxyStatsHeaders", arg0)
	ret0, _ := ret[0].(map[string]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetProxyStatsHeaders indicates an expected call of GetProxyStatsHeaders.
func (mr *MockMeshCatalogerMockRecorder) GetProxyStatsHeaders(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetProxyStatsHeaders", reflect.TypeOf((*MockMeshCataloger)(nil).GetProxyStatsHeaders), arg0)
}

// GetResolvableEndpointsForService mocks base method.
func (m *MockMeshCataloger) GetResolvableEndpointsForService(arg0 service.MeshService) []endpoint.Endpoint {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetResolvableEndpointsForService", arg0)
	ret0, _ := ret[0].([]endpoint.Endpoint)
	return ret0
}

// GetResolvableEndpointsForService indicates an expected call of GetResolvableEndpointsForService.
func (mr *MockMeshCatalogerMockRecorder) GetResolvableEndpointsForService(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetResolvableEndpointsForService", reflect.TypeOf((*MockMeshCataloger)(nil).GetResolvableEndpointsForService), arg0)
}

// GetSecret mocks base method.
func (m *MockMeshCataloger) GetSecret(arg0, arg1 string) *models.Secret {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSecret", arg0, arg1)
	ret0, _ := ret[0].(*models.Secret)
	return ret0
}

// GetSecret indicates an expected call of GetSecret.
func (mr *MockMeshCatalogerMockRecorder) GetSecret(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSecret", reflect.TypeOf((*MockMeshCataloger)(nil).GetSecret), arg0, arg1)
}

// GetServicesForServiceIdentity mocks base method.
func (m *MockMeshCataloger) GetServicesForServiceIdentity(arg0 identity.ServiceIdentity) []service.MeshService {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetServicesForServiceIdentity", arg0)
	ret0, _ := ret[0].([]service.MeshService)
	return ret0
}

// GetServicesForServiceIdentity indicates an expected call of GetServicesForServiceIdentity.
func (mr *MockMeshCatalogerMockRecorder) GetServicesForServiceIdentity(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetServicesForServiceIdentity", reflect.TypeOf((*MockMeshCataloger)(nil).GetServicesForServiceIdentity), arg0)
}

// GetTCPRoute mocks base method.
func (m *MockMeshCataloger) GetTCPRoute(arg0 string) *v1alpha4.TCPRoute {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTCPRoute", arg0)
	ret0, _ := ret[0].(*v1alpha4.TCPRoute)
	return ret0
}

// GetTCPRoute indicates an expected call of GetTCPRoute.
func (mr *MockMeshCatalogerMockRecorder) GetTCPRoute(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTCPRoute", reflect.TypeOf((*MockMeshCataloger)(nil).GetTCPRoute), arg0)
}

// GetUpstreamTrafficSetting mocks base method.
func (m *MockMeshCataloger) GetUpstreamTrafficSetting(arg0 *types.NamespacedName) *v1alpha1.UpstreamTrafficSetting {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUpstreamTrafficSetting", arg0)
	ret0, _ := ret[0].(*v1alpha1.UpstreamTrafficSetting)
	return ret0
}

// GetUpstreamTrafficSetting indicates an expected call of GetUpstreamTrafficSetting.
func (mr *MockMeshCatalogerMockRecorder) GetUpstreamTrafficSetting(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUpstreamTrafficSetting", reflect.TypeOf((*MockMeshCataloger)(nil).GetUpstreamTrafficSetting), arg0)
}

// GetUpstreamTrafficSettingByHost mocks base method.
func (m *MockMeshCataloger) GetUpstreamTrafficSettingByHost(arg0 string) *v1alpha1.UpstreamTrafficSetting {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUpstreamTrafficSettingByHost", arg0)
	ret0, _ := ret[0].(*v1alpha1.UpstreamTrafficSetting)
	return ret0
}

// GetUpstreamTrafficSettingByHost indicates an expected call of GetUpstreamTrafficSettingByHost.
func (mr *MockMeshCatalogerMockRecorder) GetUpstreamTrafficSettingByHost(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUpstreamTrafficSettingByHost", reflect.TypeOf((*MockMeshCataloger)(nil).GetUpstreamTrafficSettingByHost), arg0)
}

// GetUpstreamTrafficSettingByNamespace mocks base method.
func (m *MockMeshCataloger) GetUpstreamTrafficSettingByNamespace(arg0 *types.NamespacedName) *v1alpha1.UpstreamTrafficSetting {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUpstreamTrafficSettingByNamespace", arg0)
	ret0, _ := ret[0].(*v1alpha1.UpstreamTrafficSetting)
	return ret0
}

// GetUpstreamTrafficSettingByNamespace indicates an expected call of GetUpstreamTrafficSettingByNamespace.
func (mr *MockMeshCatalogerMockRecorder) GetUpstreamTrafficSettingByNamespace(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUpstreamTrafficSettingByNamespace", reflect.TypeOf((*MockMeshCataloger)(nil).GetUpstreamTrafficSettingByNamespace), arg0)
}

// GetUpstreamTrafficSettingByService mocks base method.
func (m *MockMeshCataloger) GetUpstreamTrafficSettingByService(arg0 *service.MeshService) *v1alpha1.UpstreamTrafficSetting {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUpstreamTrafficSettingByService", arg0)
	ret0, _ := ret[0].(*v1alpha1.UpstreamTrafficSetting)
	return ret0
}

// GetUpstreamTrafficSettingByService indicates an expected call of GetUpstreamTrafficSettingByService.
func (mr *MockMeshCatalogerMockRecorder) GetUpstreamTrafficSettingByService(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUpstreamTrafficSettingByService", reflect.TypeOf((*MockMeshCataloger)(nil).GetUpstreamTrafficSettingByService), arg0)
}

// IsMetricsEnabled mocks base method.
func (m *MockMeshCataloger) IsMetricsEnabled(arg0 *models.Proxy) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsMetricsEnabled", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// IsMetricsEnabled indicates an expected call of IsMetricsEnabled.
func (mr *MockMeshCatalogerMockRecorder) IsMetricsEnabled(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsMetricsEnabled", reflect.TypeOf((*MockMeshCataloger)(nil).IsMetricsEnabled), arg0)
}

// IsMonitoredNamespace mocks base method.
func (m *MockMeshCataloger) IsMonitoredNamespace(arg0 string) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsMonitoredNamespace", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsMonitoredNamespace indicates an expected call of IsMonitoredNamespace.
func (mr *MockMeshCatalogerMockRecorder) IsMonitoredNamespace(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsMonitoredNamespace", reflect.TypeOf((*MockMeshCataloger)(nil).IsMonitoredNamespace), arg0)
}

// ListAllowedUpstreamEndpointsForService mocks base method.
func (m *MockMeshCataloger) ListAllowedUpstreamEndpointsForService(arg0 identity.ServiceIdentity, arg1 service.MeshService) []endpoint.Endpoint {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListAllowedUpstreamEndpointsForService", arg0, arg1)
	ret0, _ := ret[0].([]endpoint.Endpoint)
	return ret0
}

// ListAllowedUpstreamEndpointsForService indicates an expected call of ListAllowedUpstreamEndpointsForService.
func (mr *MockMeshCatalogerMockRecorder) ListAllowedUpstreamEndpointsForService(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListAllowedUpstreamEndpointsForService", reflect.TypeOf((*MockMeshCataloger)(nil).ListAllowedUpstreamEndpointsForService), arg0, arg1)
}

// ListEgressPolicies mocks base method.
func (m *MockMeshCataloger) ListEgressPolicies() []*v1alpha1.Egress {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListEgressPolicies")
	ret0, _ := ret[0].([]*v1alpha1.Egress)
	return ret0
}

// ListEgressPolicies indicates an expected call of ListEgressPolicies.
func (mr *MockMeshCatalogerMockRecorder) ListEgressPolicies() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListEgressPolicies", reflect.TypeOf((*MockMeshCataloger)(nil).ListEgressPolicies))
}

// ListEgressPoliciesForServiceAccount mocks base method.
func (m *MockMeshCataloger) ListEgressPoliciesForServiceAccount(arg0 identity.K8sServiceAccount) []*v1alpha1.Egress {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListEgressPoliciesForServiceAccount", arg0)
	ret0, _ := ret[0].([]*v1alpha1.Egress)
	return ret0
}

// ListEgressPoliciesForServiceAccount indicates an expected call of ListEgressPoliciesForServiceAccount.
func (mr *MockMeshCatalogerMockRecorder) ListEgressPoliciesForServiceAccount(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListEgressPoliciesForServiceAccount", reflect.TypeOf((*MockMeshCataloger)(nil).ListEgressPoliciesForServiceAccount), arg0)
}

// ListEndpointsForIdentity mocks base method.
func (m *MockMeshCataloger) ListEndpointsForIdentity(arg0 identity.ServiceIdentity) []endpoint.Endpoint {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListEndpointsForIdentity", arg0)
	ret0, _ := ret[0].([]endpoint.Endpoint)
	return ret0
}

// ListEndpointsForIdentity indicates an expected call of ListEndpointsForIdentity.
func (mr *MockMeshCatalogerMockRecorder) ListEndpointsForIdentity(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListEndpointsForIdentity", reflect.TypeOf((*MockMeshCataloger)(nil).ListEndpointsForIdentity), arg0)
}

// ListEndpointsForService mocks base method.
func (m *MockMeshCataloger) ListEndpointsForService(arg0 service.MeshService) []endpoint.Endpoint {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListEndpointsForService", arg0)
	ret0, _ := ret[0].([]endpoint.Endpoint)
	return ret0
}

// ListEndpointsForService indicates an expected call of ListEndpointsForService.
func (mr *MockMeshCatalogerMockRecorder) ListEndpointsForService(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListEndpointsForService", reflect.TypeOf((*MockMeshCataloger)(nil).ListEndpointsForService), arg0)
}

// ListHTTPTrafficSpecs mocks base method.
func (m *MockMeshCataloger) ListHTTPTrafficSpecs() []*v1alpha4.HTTPRouteGroup {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListHTTPTrafficSpecs")
	ret0, _ := ret[0].([]*v1alpha4.HTTPRouteGroup)
	return ret0
}

// ListHTTPTrafficSpecs indicates an expected call of ListHTTPTrafficSpecs.
func (mr *MockMeshCatalogerMockRecorder) ListHTTPTrafficSpecs() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListHTTPTrafficSpecs", reflect.TypeOf((*MockMeshCataloger)(nil).ListHTTPTrafficSpecs))
}

// ListInboundServiceIdentities mocks base method.
func (m *MockMeshCataloger) ListInboundServiceIdentities(arg0 identity.ServiceIdentity) []identity.ServiceIdentity {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListInboundServiceIdentities", arg0)
	ret0, _ := ret[0].([]identity.ServiceIdentity)
	return ret0
}

// ListInboundServiceIdentities indicates an expected call of ListInboundServiceIdentities.
func (mr *MockMeshCatalogerMockRecorder) ListInboundServiceIdentities(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListInboundServiceIdentities", reflect.TypeOf((*MockMeshCataloger)(nil).ListInboundServiceIdentities), arg0)
}

// ListInboundTrafficTargetsWithRoutes mocks base method.
func (m *MockMeshCataloger) ListInboundTrafficTargetsWithRoutes(arg0 identity.ServiceIdentity) ([]trafficpolicy.TrafficTargetWithRoutes, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListInboundTrafficTargetsWithRoutes", arg0)
	ret0, _ := ret[0].([]trafficpolicy.TrafficTargetWithRoutes)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListInboundTrafficTargetsWithRoutes indicates an expected call of ListInboundTrafficTargetsWithRoutes.
func (mr *MockMeshCatalogerMockRecorder) ListInboundTrafficTargetsWithRoutes(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListInboundTrafficTargetsWithRoutes", reflect.TypeOf((*MockMeshCataloger)(nil).ListInboundTrafficTargetsWithRoutes), arg0)
}

// ListIngressBackendPolicies mocks base method.
func (m *MockMeshCataloger) ListIngressBackendPolicies() []*v1alpha1.IngressBackend {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListIngressBackendPolicies")
	ret0, _ := ret[0].([]*v1alpha1.IngressBackend)
	return ret0
}

// ListIngressBackendPolicies indicates an expected call of ListIngressBackendPolicies.
func (mr *MockMeshCatalogerMockRecorder) ListIngressBackendPolicies() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListIngressBackendPolicies", reflect.TypeOf((*MockMeshCataloger)(nil).ListIngressBackendPolicies))
}

// ListMeshRootCertificates mocks base method.
func (m *MockMeshCataloger) ListMeshRootCertificates() ([]*v1alpha2.MeshRootCertificate, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListMeshRootCertificates")
	ret0, _ := ret[0].([]*v1alpha2.MeshRootCertificate)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListMeshRootCertificates indicates an expected call of ListMeshRootCertificates.
func (mr *MockMeshCatalogerMockRecorder) ListMeshRootCertificates() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListMeshRootCertificates", reflect.TypeOf((*MockMeshCataloger)(nil).ListMeshRootCertificates))
}

// ListNamespaces mocks base method.
func (m *MockMeshCataloger) ListNamespaces() ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListNamespaces")
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListNamespaces indicates an expected call of ListNamespaces.
func (mr *MockMeshCatalogerMockRecorder) ListNamespaces() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListNamespaces", reflect.TypeOf((*MockMeshCataloger)(nil).ListNamespaces))
}

// ListOutboundServiceIdentities mocks base method.
func (m *MockMeshCataloger) ListOutboundServiceIdentities(arg0 identity.ServiceIdentity) []identity.ServiceIdentity {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListOutboundServiceIdentities", arg0)
	ret0, _ := ret[0].([]identity.ServiceIdentity)
	return ret0
}

// ListOutboundServiceIdentities indicates an expected call of ListOutboundServiceIdentities.
func (mr *MockMeshCatalogerMockRecorder) ListOutboundServiceIdentities(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListOutboundServiceIdentities", reflect.TypeOf((*MockMeshCataloger)(nil).ListOutboundServiceIdentities), arg0)
}

// ListOutboundServicesForIdentity mocks base method.
func (m *MockMeshCataloger) ListOutboundServicesForIdentity(arg0 identity.ServiceIdentity) []service.MeshService {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListOutboundServicesForIdentity", arg0)
	ret0, _ := ret[0].([]service.MeshService)
	return ret0
}

// ListOutboundServicesForIdentity indicates an expected call of ListOutboundServicesForIdentity.
func (mr *MockMeshCatalogerMockRecorder) ListOutboundServicesForIdentity(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListOutboundServicesForIdentity", reflect.TypeOf((*MockMeshCataloger)(nil).ListOutboundServicesForIdentity), arg0)
}

// ListRetryPolicies mocks base method.
func (m *MockMeshCataloger) ListRetryPolicies() []*v1alpha1.Retry {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRetryPolicies")
	ret0, _ := ret[0].([]*v1alpha1.Retry)
	return ret0
}

// ListRetryPolicies indicates an expected call of ListRetryPolicies.
func (mr *MockMeshCatalogerMockRecorder) ListRetryPolicies() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRetryPolicies", reflect.TypeOf((*MockMeshCataloger)(nil).ListRetryPolicies))
}

// ListRetryPoliciesForServiceAccount mocks base method.
func (m *MockMeshCataloger) ListRetryPoliciesForServiceAccount(arg0 identity.K8sServiceAccount) []*v1alpha1.Retry {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRetryPoliciesForServiceAccount", arg0)
	ret0, _ := ret[0].([]*v1alpha1.Retry)
	return ret0
}

// ListRetryPoliciesForServiceAccount indicates an expected call of ListRetryPoliciesForServiceAccount.
func (mr *MockMeshCatalogerMockRecorder) ListRetryPoliciesForServiceAccount(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRetryPoliciesForServiceAccount", reflect.TypeOf((*MockMeshCataloger)(nil).ListRetryPoliciesForServiceAccount), arg0)
}

// ListSecrets mocks base method.
func (m *MockMeshCataloger) ListSecrets() []*models.Secret {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListSecrets")
	ret0, _ := ret[0].([]*models.Secret)
	return ret0
}

// ListSecrets indicates an expected call of ListSecrets.
func (mr *MockMeshCatalogerMockRecorder) ListSecrets() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListSecrets", reflect.TypeOf((*MockMeshCataloger)(nil).ListSecrets))
}

// ListServiceAccountsFromTrafficTargets mocks base method.
func (m *MockMeshCataloger) ListServiceAccountsFromTrafficTargets() []identity.K8sServiceAccount {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListServiceAccountsFromTrafficTargets")
	ret0, _ := ret[0].([]identity.K8sServiceAccount)
	return ret0
}

// ListServiceAccountsFromTrafficTargets indicates an expected call of ListServiceAccountsFromTrafficTargets.
func (mr *MockMeshCatalogerMockRecorder) ListServiceAccountsFromTrafficTargets() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListServiceAccountsFromTrafficTargets", reflect.TypeOf((*MockMeshCataloger)(nil).ListServiceAccountsFromTrafficTargets))
}

// ListServiceIdentitiesForService mocks base method.
func (m *MockMeshCataloger) ListServiceIdentitiesForService(arg0, arg1 string) ([]identity.ServiceIdentity, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListServiceIdentitiesForService", arg0, arg1)
	ret0, _ := ret[0].([]identity.ServiceIdentity)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListServiceIdentitiesForService indicates an expected call of ListServiceIdentitiesForService.
func (mr *MockMeshCatalogerMockRecorder) ListServiceIdentitiesForService(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListServiceIdentitiesForService", reflect.TypeOf((*MockMeshCataloger)(nil).ListServiceIdentitiesForService), arg0, arg1)
}

// ListServices mocks base method.
func (m *MockMeshCataloger) ListServices() []service.MeshService {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListServices")
	ret0, _ := ret[0].([]service.MeshService)
	return ret0
}

// ListServices indicates an expected call of ListServices.
func (mr *MockMeshCatalogerMockRecorder) ListServices() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListServices", reflect.TypeOf((*MockMeshCataloger)(nil).ListServices))
}

// ListServicesForProxy mocks base method.
func (m *MockMeshCataloger) ListServicesForProxy(arg0 *models.Proxy) ([]service.MeshService, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListServicesForProxy", arg0)
	ret0, _ := ret[0].([]service.MeshService)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListServicesForProxy indicates an expected call of ListServicesForProxy.
func (mr *MockMeshCatalogerMockRecorder) ListServicesForProxy(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListServicesForProxy", reflect.TypeOf((*MockMeshCataloger)(nil).ListServicesForProxy), arg0)
}

// ListTCPTrafficSpecs mocks base method.
func (m *MockMeshCataloger) ListTCPTrafficSpecs() []*v1alpha4.TCPRoute {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListTCPTrafficSpecs")
	ret0, _ := ret[0].([]*v1alpha4.TCPRoute)
	return ret0
}

// ListTCPTrafficSpecs indicates an expected call of ListTCPTrafficSpecs.
func (mr *MockMeshCatalogerMockRecorder) ListTCPTrafficSpecs() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListTCPTrafficSpecs", reflect.TypeOf((*MockMeshCataloger)(nil).ListTCPTrafficSpecs))
}

// ListTrafficSplits mocks base method.
func (m *MockMeshCataloger) ListTrafficSplits() []*v1alpha20.TrafficSplit {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListTrafficSplits")
	ret0, _ := ret[0].([]*v1alpha20.TrafficSplit)
	return ret0
}

// ListTrafficSplits indicates an expected call of ListTrafficSplits.
func (mr *MockMeshCatalogerMockRecorder) ListTrafficSplits() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListTrafficSplits", reflect.TypeOf((*MockMeshCataloger)(nil).ListTrafficSplits))
}

// ListTrafficTargets mocks base method.
func (m *MockMeshCataloger) ListTrafficTargets() []*v1alpha3.TrafficTarget {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListTrafficTargets")
	ret0, _ := ret[0].([]*v1alpha3.TrafficTarget)
	return ret0
}

// ListTrafficTargets indicates an expected call of ListTrafficTargets.
func (mr *MockMeshCatalogerMockRecorder) ListTrafficTargets() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListTrafficTargets", reflect.TypeOf((*MockMeshCataloger)(nil).ListTrafficTargets))
}

// ListUpstreamTrafficSettings mocks base method.
func (m *MockMeshCataloger) ListUpstreamTrafficSettings() []*v1alpha1.UpstreamTrafficSetting {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListUpstreamTrafficSettings")
	ret0, _ := ret[0].([]*v1alpha1.UpstreamTrafficSetting)
	return ret0
}

// ListUpstreamTrafficSettings indicates an expected call of ListUpstreamTrafficSettings.
func (mr *MockMeshCatalogerMockRecorder) ListUpstreamTrafficSettings() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListUpstreamTrafficSettings", reflect.TypeOf((*MockMeshCataloger)(nil).ListUpstreamTrafficSettings))
}

// UpdateIngressBackendStatus mocks base method.
func (m *MockMeshCataloger) UpdateIngressBackendStatus(arg0 *v1alpha1.IngressBackend) (*v1alpha1.IngressBackend, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateIngressBackendStatus", arg0)
	ret0, _ := ret[0].(*v1alpha1.IngressBackend)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateIngressBackendStatus indicates an expected call of UpdateIngressBackendStatus.
func (mr *MockMeshCatalogerMockRecorder) UpdateIngressBackendStatus(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateIngressBackendStatus", reflect.TypeOf((*MockMeshCataloger)(nil).UpdateIngressBackendStatus), arg0)
}

// UpdateMeshRootCertificate mocks base method.
func (m *MockMeshCataloger) UpdateMeshRootCertificate(arg0 *v1alpha2.MeshRootCertificate) (*v1alpha2.MeshRootCertificate, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateMeshRootCertificate", arg0)
	ret0, _ := ret[0].(*v1alpha2.MeshRootCertificate)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateMeshRootCertificate indicates an expected call of UpdateMeshRootCertificate.
func (mr *MockMeshCatalogerMockRecorder) UpdateMeshRootCertificate(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateMeshRootCertificate", reflect.TypeOf((*MockMeshCataloger)(nil).UpdateMeshRootCertificate), arg0)
}

// UpdateMeshRootCertificateStatus mocks base method.
func (m *MockMeshCataloger) UpdateMeshRootCertificateStatus(arg0 *v1alpha2.MeshRootCertificate) (*v1alpha2.MeshRootCertificate, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateMeshRootCertificateStatus", arg0)
	ret0, _ := ret[0].(*v1alpha2.MeshRootCertificate)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateMeshRootCertificateStatus indicates an expected call of UpdateMeshRootCertificateStatus.
func (mr *MockMeshCatalogerMockRecorder) UpdateMeshRootCertificateStatus(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateMeshRootCertificateStatus", reflect.TypeOf((*MockMeshCataloger)(nil).UpdateMeshRootCertificateStatus), arg0)
}

// UpdateSecret mocks base method.
func (m *MockMeshCataloger) UpdateSecret(arg0 context.Context, arg1 *models.Secret) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateSecret", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateSecret indicates an expected call of UpdateSecret.
func (mr *MockMeshCatalogerMockRecorder) UpdateSecret(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateSecret", reflect.TypeOf((*MockMeshCataloger)(nil).UpdateSecret), arg0, arg1)
}

// UpdateUpstreamTrafficSettingStatus mocks base method.
func (m *MockMeshCataloger) UpdateUpstreamTrafficSettingStatus(arg0 *v1alpha1.UpstreamTrafficSetting) (*v1alpha1.UpstreamTrafficSetting, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateUpstreamTrafficSettingStatus", arg0)
	ret0, _ := ret[0].(*v1alpha1.UpstreamTrafficSetting)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateUpstreamTrafficSettingStatus indicates an expected call of UpdateUpstreamTrafficSettingStatus.
func (mr *MockMeshCatalogerMockRecorder) UpdateUpstreamTrafficSettingStatus(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateUpstreamTrafficSettingStatus", reflect.TypeOf((*MockMeshCataloger)(nil).UpdateUpstreamTrafficSettingStatus), arg0)
}

// VerifyProxy mocks base method.
func (m *MockMeshCataloger) VerifyProxy(arg0 *models.Proxy) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "VerifyProxy", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// VerifyProxy indicates an expected call of VerifyProxy.
func (mr *MockMeshCatalogerMockRecorder) VerifyProxy(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "VerifyProxy", reflect.TypeOf((*MockMeshCataloger)(nil).VerifyProxy), arg0)
}
