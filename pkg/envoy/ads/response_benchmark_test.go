package ads

import (
	"context"
	"fmt"
	"testing"

	mapset "github.com/deckarep/golang-set"
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sClientFake "k8s.io/client-go/kubernetes/fake"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/k8s/informers"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/policy"
	"github.com/openservicemesh/osm/pkg/providers/kube"
	"github.com/openservicemesh/osm/pkg/signals"
	"github.com/openservicemesh/osm/pkg/smi"

	"github.com/openservicemesh/osm/pkg/certificate"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	configFake "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
	policyFake "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned/fake"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

var (
	proxy           *envoy.Proxy
	server          xds_discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer
	osmConfigurator *configurator.Client
	adsServer       *Server
)

func setupTestServer(b *testing.B) {
	stop := signals.RegisterExitHandlers()
	msgBroker := messaging.NewBroker(stop)
	kubeClient := k8sClientFake.NewSimpleClientset()
	configClient := configFake.NewSimpleClientset()
	policyClient := policyFake.NewSimpleClientset()
	informerCollection, err := informers.NewInformerCollection(tests.MeshName, stop,
		informers.WithKubeClient(kubeClient),
		informers.WithConfigClient(configClient, tests.OsmMeshConfigName, tests.OsmNamespace),
	)
	if err != nil {
		b.Fatalf("Failed to create informer collection: %s", err)
	}
	kubeController := k8s.NewKubernetesController(informerCollection, policyClient, msgBroker)
	policyController := policy.NewPolicyController(informerCollection, kubeController, msgBroker)
	osmConfigurator = configurator.NewConfigurator(informerCollection, tests.OsmNamespace, tests.OsmMeshConfigName, msgBroker)
	kubeProvider := kube.NewClient(kubeController, osmConfigurator)

	meshConfig := configv1alpha2.MeshConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "config.openservicemesh.io",
			Kind:       "MeshConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tests.OsmNamespace,
			Name:      tests.OsmMeshConfigName,
		}, Spec: configv1alpha2.MeshConfigSpec{
			Certificate: configv1alpha2.CertificateSpec{
				CertKeyBitSize:              2048,
				ServiceCertValidityDuration: "1h",
			},
			Traffic: configv1alpha2.TrafficSpec{
				EnableEgress:                      true,
				EnablePermissiveTrafficPolicyMode: true,
			},
			Observability: configv1alpha2.ObservabilitySpec{
				EnableDebugServer: false,
				Tracing: configv1alpha2.TracingSpec{
					Enable: false,
				},
			},
			FeatureFlags: configv1alpha2.FeatureFlags{
				EnableWASMStats:    false,
				EnableEgressPolicy: false,
			},
		},
	}
	_, err = configClient.ConfigV1alpha2().MeshConfigs(tests.OsmNamespace).Create(context.Background(), &meshConfig, metav1.CreateOptions{})
	if err != nil {
		b.Fatalf("Failed to create mesh config: %v", err)
	}

	certManager, err := certificate.FakeCertManager()
	if err != nil {
		b.Fatalf("Failed to create fake cert manager: %v", err)
	}

	// --- setup
	namespace := tests.Namespace
	proxyService := service.MeshService{
		Name:      tests.BookstoreV1ServiceName,
		Namespace: namespace,
	}
	proxySvcAccount := tests.BookstoreServiceAccount

	certPEM, _ := certManager.IssueCertificate(proxySvcAccount.ToServiceIdentity().String(), certificate.Service)
	cert, _ := certificate.DecodePEMCertificate(certPEM.GetCertificateChain())
	server, _ = tests.NewFakeXDSServer(cert, nil, nil)

	proxyUUID := uuid.New()
	labels := map[string]string{constants.EnvoyUniqueIDLabelName: proxyUUID.String()}
	meshSpec := smi.NewSMIClient(informerCollection, tests.OsmNamespace, kubeController, msgBroker)
	mc := catalog.NewMeshCatalog(
		kubeController,
		meshSpec,
		certManager,
		policyController,
		stop,
		osmConfigurator,
		[]service.Provider{kubeProvider},
		[]endpoint.Provider{kubeProvider},
		msgBroker,
	)

	proxyRegistry := registry.NewProxyRegistry(registry.ExplicitProxyServiceMapper(func(*envoy.Proxy) ([]service.MeshService, error) {
		return nil, nil
	}), nil)

	pod := tests.NewPodFixture(namespace, fmt.Sprintf("pod-0-%s", proxyUUID), tests.BookstoreServiceAccountName, tests.PodLabels)
	pod.Labels[constants.EnvoyUniqueIDLabelName] = proxyUUID.String()
	_, err = kubeClient.CoreV1().Pods(namespace).Create(context.Background(), pod, metav1.CreateOptions{})
	if err != nil {
		b.Fatalf("Failed to create pod: %v", err)
	}

	// monitor namespace
	nsObj := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   namespace,
			Labels: map[string]string{constants.OSMKubeResourceMonitorAnnotation: tests.MeshName},
		},
	}
	if err := informerCollection.Add(informers.InformerKeyNamespace, &nsObj, &testing.T{}); err != nil {
		b.Fatalf("Failed to add namespace to informer collection: %s", err)
	}

	svc := tests.NewServiceFixture(proxyService.Name, namespace, labels)
	_, err = kubeClient.CoreV1().Services(namespace).Create(context.Background(), svc, metav1.CreateOptions{})
	if err != nil {
		b.Fatalf("Failed to create service: %v", err)
	}

	proxy = envoy.NewProxy(envoy.KindSidecar, proxyUUID, proxySvcAccount.ToServiceIdentity(), nil)

	adsServer = NewADSServer(mc, proxyRegistry, true, tests.Namespace, osmConfigurator, certManager, kubeController, nil)
}

func BenchmarkSendXDSResponse(b *testing.B) {
	// TODO(allenlsy): Add EDS to the list
	testingXdsTypes := []envoy.TypeURI{
		envoy.TypeLDS,
		envoy.TypeSDS,
		envoy.TypeRDS,
		envoy.TypeCDS,
	}

	if err := logger.SetLogLevel("error"); err != nil {
		b.Logf("Failed to set log level to error: %s", err)
	}

	for _, xdsType := range testingXdsTypes {
		b.Run(string(xdsType), func(b *testing.B) {
			setupTestServer(b)

			// Set subscribed resources
			proxy.SetSubscribedResources(xdsType, mapset.NewSetWith("service-cert:default/bookstore", "root-cert-for-mtls-inbound:default/bookstore|80"))

			b.ResetTimer()
			b.StartTimer()
			for i := 0; i < b.N; i++ {
				if err := adsServer.sendResponse(proxy, &server, nil, osmConfigurator, xdsType); err != nil {
					b.Fatalf("Failed to send response: %s", err)
				}
			}
			b.StopTimer()
		})
	}
}
