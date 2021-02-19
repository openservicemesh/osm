package eds

import (
	"context"
	"fmt"
	"net"
	"testing"

	xds_endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes"
	"github.com/google/uuid"
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	tassert "github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/envoy"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/tests"
)

func getProxy(kubeClient kubernetes.Interface) (*envoy.Proxy, error) {
	podLabels := map[string]string{
		tests.SelectorKey:                tests.BookbuyerService.Name,
		constants.EnvoyUniqueIDLabelName: tests.ProxyUUID,
	}
	if _, err := tests.MakePod(kubeClient, tests.Namespace, tests.BookbuyerServiceName, tests.BookbuyerServiceAccountName, podLabels); err != nil {
		return nil, err
	}

	selectors := map[string]string{
		tests.SelectorKey: tests.BookbuyerServiceName,
	}
	if _, err := tests.MakeService(kubeClient, tests.BookbuyerServiceName, selectors); err != nil {
		return nil, err
	}

	for _, svcName := range []string{tests.BookstoreApexServiceName, tests.BookstoreV1ServiceName, tests.BookstoreV2ServiceName} {
		selectors := map[string]string{
			tests.SelectorKey: "bookstore",
		}
		if _, err := tests.MakeService(kubeClient, svcName, selectors); err != nil {
			return nil, err
		}
	}

	certCommonName := certificate.CommonName(fmt.Sprintf("%s.%s.%s", tests.ProxyUUID, tests.BookbuyerServiceAccountName, tests.Namespace))
	certSerialNumber := certificate.SerialNumber("123456")
	proxy := envoy.NewProxy(certCommonName, certSerialNumber, nil)
	return proxy, nil
}

func TestEndpointConfiguration(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	kubeClient := testclient.NewSimpleClientset()
	meshCatalog := catalog.NewFakeMeshCatalog(kubeClient)

	proxy, err := getProxy(kubeClient)
	assert.Empty(err)
	assert.NotNil(meshCatalog)
	assert.NotNil(proxy)

	actual, err := NewResponse(meshCatalog, proxy, nil, mockConfigurator, nil)
	assert.Nil(err)
	assert.NotNil(actual)

	// There are 3 endpoints configured based on the configuration:
	// 1. Bookstore
	// 2. Bookstore-v1
	// 3. Bookstore-v2
	assert.Len(actual.Resources, 3)

	loadAssignment := xds_endpoint.ClusterLoadAssignment{}

	// validating an endpoint
	err = ptypes.UnmarshalAny(actual.Resources[0], &loadAssignment)
	assert.Nil(err)
	assert.Len(loadAssignment.Endpoints, 1)
}

func TestGetEndpointsForProxy(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		name                            string
		proxyIdentity                   service.K8sServiceAccount
		trafficTargets                  []*access.TrafficTarget
		allowedServiceAccounts          []service.K8sServiceAccount
		services                        []service.MeshService
		outboundServices                map[service.K8sServiceAccount][]service.MeshService
		outboundServiceEndpoints        map[service.MeshService][]endpoint.Endpoint
		outboundServiceAccountEndpoints map[service.K8sServiceAccount]map[service.MeshService][]endpoint.Endpoint
		expectedEndpoints               map[service.MeshService][]endpoint.Endpoint
	}{
		{
			name: `Traffic target defined for bookstore ServiceAccount.
			This service account has bookstore-v1 service which has one endpoint.
			Hence one endpoint for bookstore-v1 should be in the expected list`,
			proxyIdentity:          tests.BookbuyerServiceAccount,
			trafficTargets:         []*access.TrafficTarget{&tests.TrafficTarget},
			allowedServiceAccounts: []service.K8sServiceAccount{tests.BookstoreServiceAccount},
			services:               []service.MeshService{tests.BookstoreV1Service},
			outboundServices: map[service.K8sServiceAccount][]service.MeshService{
				tests.BookstoreServiceAccount: {tests.BookstoreV1Service},
			},
			outboundServiceEndpoints: map[service.MeshService][]endpoint.Endpoint{
				tests.BookstoreV1Service: {tests.Endpoint},
			},
			outboundServiceAccountEndpoints: map[service.K8sServiceAccount]map[service.MeshService][]endpoint.Endpoint{
				tests.BookstoreServiceAccount: {tests.BookstoreV1Service: {tests.Endpoint}},
			},
			expectedEndpoints: map[service.MeshService][]endpoint.Endpoint{
				tests.BookstoreV1Service: {tests.Endpoint},
			},
		},
		{
			name: `Traffic target defined for bookstore ServiceAccount.
			This service account has bookstore-v1 service which has two endpoints,
			but endpoint 9.9.9.9 is associated with a pod having service account bookstore-v2.
			Hence this endpoint (9.9.9.9) shouldn't be in bookstore-v1's expected list`,
			proxyIdentity:          tests.BookbuyerServiceAccount,
			trafficTargets:         []*access.TrafficTarget{&tests.TrafficTarget},
			allowedServiceAccounts: []service.K8sServiceAccount{tests.BookstoreServiceAccount},
			services:               []service.MeshService{tests.BookstoreV1Service},
			outboundServices: map[service.K8sServiceAccount][]service.MeshService{
				tests.BookstoreServiceAccount: {tests.BookstoreV1Service},
			},
			outboundServiceEndpoints: map[service.MeshService][]endpoint.Endpoint{
				tests.BookstoreV1Service: {tests.Endpoint, {
					IP:   net.ParseIP("9.9.9.9"),
					Port: endpoint.Port(tests.ServicePort),
				}},
			},
			outboundServiceAccountEndpoints: map[service.K8sServiceAccount]map[service.MeshService][]endpoint.Endpoint{
				tests.BookstoreServiceAccount: {tests.BookstoreV1Service: {tests.Endpoint}},
			},
			expectedEndpoints: map[service.MeshService][]endpoint.Endpoint{
				tests.BookstoreV1Service: {tests.Endpoint},
			},
		},
		{
			name: `Traffic target defined for bookstore and bookstore-v2 ServiceAccount.
			Hence one endpoint should be in bookstore-v1's and bookstore-v2's expected list`,
			proxyIdentity:          tests.BookbuyerServiceAccount,
			trafficTargets:         []*access.TrafficTarget{&tests.TrafficTarget, &tests.BookstoreV2TrafficTarget},
			allowedServiceAccounts: []service.K8sServiceAccount{tests.BookstoreServiceAccount, tests.BookstoreV2ServiceAccount},
			services:               []service.MeshService{tests.BookstoreV1Service, tests.BookstoreV2Service},
			outboundServices: map[service.K8sServiceAccount][]service.MeshService{
				tests.BookstoreServiceAccount:   {tests.BookstoreV1Service},
				tests.BookstoreV2ServiceAccount: {tests.BookstoreV2Service},
			},
			outboundServiceEndpoints: map[service.MeshService][]endpoint.Endpoint{
				tests.BookstoreV1Service: {tests.Endpoint},
				tests.BookstoreV2Service: {endpoint.Endpoint{
					IP:   net.ParseIP("9.9.9.9"),
					Port: endpoint.Port(tests.ServicePort),
				}},
			},
			outboundServiceAccountEndpoints: map[service.K8sServiceAccount]map[service.MeshService][]endpoint.Endpoint{
				tests.BookstoreServiceAccount: {tests.BookstoreV1Service: {tests.Endpoint}},
				tests.BookstoreV2ServiceAccount: {tests.BookstoreV2Service: {endpoint.Endpoint{
					IP:   net.ParseIP("9.9.9.9"),
					Port: endpoint.Port(tests.ServicePort),
				}}},
			},
			expectedEndpoints: map[service.MeshService][]endpoint.Endpoint{
				tests.BookstoreV1Service: {tests.Endpoint},
				tests.BookstoreV2Service: {endpoint.Endpoint{
					IP:   net.ParseIP("9.9.9.9"),
					Port: endpoint.Port(tests.ServicePort),
				}},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			kubeClient := testclient.NewSimpleClientset()
			defer mockCtrl.Finish()

			mockCatalog := catalog.NewMockMeshCataloger(mockCtrl)
			mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
			mockKubeController := k8s.NewMockController(mockCtrl)
			meshSpec := smi.NewMockMeshSpec(mockCtrl)
			mockEndpointProvider := endpoint.NewMockProvider(mockCtrl)

			mockConfigurator.EXPECT().IsPermissiveTrafficPolicyMode().Return(false).AnyTimes()
			meshSpec.EXPECT().ListTrafficTargets().Return(tc.trafficTargets).AnyTimes()

			proxy, err := getProxy(kubeClient)
			assert.Empty(err)
			assert.NotNil(mockCatalog)
			assert.NotNil(proxy)

			mockEndpointProvider.EXPECT().GetID().Return("fake").AnyTimes()

			for sa, services := range tc.outboundServices {
				for _, svc := range services {
					k8sService := tests.NewServiceFixture(svc.Name, svc.Namespace, map[string]string{})
					mockKubeController.EXPECT().GetService(svc).Return(k8sService).AnyTimes()
				}
				mockEndpointProvider.EXPECT().GetServicesForServiceAccount(sa).Return(services, nil).AnyTimes()
			}

			mockCatalog.EXPECT().ListAllowedOutboundServicesForIdentity(tc.proxyIdentity).Return(tc.services).AnyTimes()

			for svc, endpoints := range tc.outboundServiceEndpoints {
				mockEndpointProvider.EXPECT().ListEndpointsForService(svc).Return(endpoints).AnyTimes()
				mockCatalog.EXPECT().ListEndpointsForService(svc).Return(endpoints, nil).AnyTimes()
			}

			mockCatalog.EXPECT().ListAllowedOutboundServiceAccounts(tc.proxyIdentity).Return(tc.allowedServiceAccounts, nil).AnyTimes()

			var pods []*v1.Pod
			for sa, services := range tc.outboundServices {
				for _, svc := range services {
					podlabels := map[string]string{
						tests.SelectorKey:                tests.SelectorValue,
						constants.EnvoyUniqueIDLabelName: uuid.New().String(),
					}
					pod := tests.NewPodFixture(tests.Namespace, svc.Name, sa.Name, podlabels)
					svcPodEndpoints := tc.outboundServiceAccountEndpoints[sa]
					var podIps []v1.PodIP
					for _, podEndpoints := range svcPodEndpoints {
						for _, ep := range podEndpoints {
							podIps = append(podIps, v1.PodIP{IP: ep.IP.String()})
						}
					}
					pod.Status.PodIPs = podIps
					pod.Spec.ServiceAccountName = sa.Name
					_, err = kubeClient.CoreV1().Pods(tests.Namespace).Create(context.TODO(), &pod, metav1.CreateOptions{})
					assert.Nil(err)
					pods = append(pods, &pod)
				}
			}
			mockKubeController.EXPECT().ListPods().Return(pods).AnyTimes()

			for sa, svcEndpoints := range tc.outboundServiceAccountEndpoints {
				for svc, endpoints := range svcEndpoints {
					mockEndpointProvider.EXPECT().ListEndpointsForIdentity(sa).Return(endpoints).AnyTimes()
					mockCatalog.EXPECT().ListAllowedEndpointsForService(tc.proxyIdentity, svc).Return(endpoints, nil).AnyTimes()
				}
			}

			actual, err := getEndpointsForProxy(mockCatalog, tc.proxyIdentity)
			assert.Nil(err)
			assert.NotNil(actual)

			assert.Len(actual, len(tc.expectedEndpoints))
			for svc, endpoints := range tc.expectedEndpoints {
				_, ok := actual[svc]
				assert.True(ok)
				assert.ElementsMatch(actual[svc], endpoints)
			}
		})
	}
}
