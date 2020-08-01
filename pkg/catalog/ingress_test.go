package catalog

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	extensionsV1beta "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/ingress"
	"github.com/openservicemesh/osm/pkg/namespace"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/smi"
)

var (
	fakeIngressService         = "fake-service"
	fakeIngressNamespace       = "ingress-ns"
	fakeIngressPort      int32 = 80

	// fakeIngressPaths is a mapping of the fake ingress resource domains to its paths
	fakeIngressPaths = map[string][]string{
		"fake1.com": []string{"/fake1-path1", "/fake1-path2"},
		"fake2.com": []string{"/fake2-path1"},
		"*":         []string{".*"},
	}
)

func newFakeMeshCatalog() *MeshCatalog {
	meshSpec := smi.NewFakeMeshSpecClient()
	cache := make(map[certificate.CommonName]certificate.Certificater)
	certManager := tresor.NewFakeCertManager(&cache, 1*time.Hour)
	ingressMonitor := ingress.NewFakeIngressMonitor()
	ingressMonitor.FakeIngresses = getFakeIngresses()
	stop := make(<-chan struct{})
	var endpointProviders []endpoint.Provider
	kubeClient := testclient.NewSimpleClientset()

	osmNamespace := "-test-osm-namespace-"
	osmConfigMapName := "-test-osm-config-map-"
	cfg := configurator.NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)

	namespaceController := namespace.NewFakeNamespaceController([]string{osmNamespace})

	return NewMeshCatalog(namespaceController, kubeClient, meshSpec, certManager, ingressMonitor, stop, cfg, endpointProviders...)
}

func getFakeIngresses() []*extensionsV1beta.Ingress {
	return []*extensionsV1beta.Ingress{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ingress-1",
				Namespace: fakeIngressNamespace,
				Annotations: map[string]string{
					constants.OSMKubeResourceMonitorAnnotation: "enabled",
				},
			},
			Spec: extensionsV1beta.IngressSpec{
				Backend: &extensionsV1beta.IngressBackend{
					ServiceName: fakeIngressService,
					ServicePort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: fakeIngressPort,
					},
				},
				Rules: []extensionsV1beta.IngressRule{
					{
						Host: "fake1.com",
						IngressRuleValue: extensionsV1beta.IngressRuleValue{
							HTTP: &extensionsV1beta.HTTPIngressRuleValue{
								Paths: []extensionsV1beta.HTTPIngressPath{
									{
										Path: "/fake1-path1",
										Backend: extensionsV1beta.IngressBackend{
											ServiceName: fakeIngressService,
											ServicePort: intstr.IntOrString{
												Type:   intstr.Int,
												IntVal: fakeIngressPort,
											},
										},
									},
									{
										Path: "/fake1-path2",
										Backend: extensionsV1beta.IngressBackend{
											ServiceName: fakeIngressService,
											ServicePort: intstr.IntOrString{
												Type:   intstr.Int,
												IntVal: fakeIngressPort,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},

		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ingress-2",
				Namespace: fakeIngressNamespace,
				Annotations: map[string]string{
					constants.OSMKubeResourceMonitorAnnotation: "enabled",
				},
			},
			Spec: extensionsV1beta.IngressSpec{
				Rules: []extensionsV1beta.IngressRule{
					{
						Host: "fake2.com",
						IngressRuleValue: extensionsV1beta.IngressRuleValue{
							HTTP: &extensionsV1beta.HTTPIngressRuleValue{
								Paths: []extensionsV1beta.HTTPIngressPath{
									{
										Path: "/fake2-path1",
										Backend: extensionsV1beta.IngressBackend{
											ServiceName: fakeIngressService,
											ServicePort: intstr.IntOrString{
												Type:   intstr.Int,
												IntVal: fakeIngressPort,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func pathContains(allowed []string, path string) bool {
	for _, p := range allowed {
		if path == p {
			return true
		}
	}
	return false
}

var _ = Describe("Test ingress route policies", func() {
	Context("Testing GetIngressRoutesPerHost", func() {
		mc := newFakeMeshCatalog()
		It("Gets the route policies per domain from multiple ingress resources corresponding to a service", func() {
			fakeService := service.NamespacedService{
				Namespace: fakeIngressNamespace,
				Service:   fakeIngressService,
			}
			domainRoutesMap, _ := mc.GetIngressRoutesPerHost(fakeService)

			for domain, routePolicies := range domainRoutesMap {
				// The number of route policies per domain is the product of the number of rules and paths per rule
				Expect(len(routePolicies)).To(Equal(len(fakeIngressPaths[domain])))
				for _, routePolicy := range routePolicies {

					// For each ingress path, all HTTP methods are allowed, which is a regex match all of '*'
					Expect(len(routePolicy.Methods)).To(Equal(1))

					Expect(routePolicy.Methods[0]).To(Equal(constants.RegexMatchAll))

					// routePolicy.Path is the path specified in the ingress resource rule. Since the same service
					// could be a backend for multiple ingress resources, we don't know which ingress resource
					// this path corresponds to just from 'domainRoutesMap'. In order to not make assumptions
					// on the implementation of 'GetIngressRoutesPerHost()', we relax the check here
					// to match on any of the ingress paths corresponding to the domain.
					Expect(pathContains(fakeIngressPaths[domain], routePolicy.PathRegex)).To(BeTrue())
				}
			}
		})

	})
})
