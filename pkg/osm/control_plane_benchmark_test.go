package osm

import (
	"context"
	"fmt"
	"time"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/google/uuid"
	"google.golang.org/grpc/peer"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sClientFake "k8s.io/client-go/kubernetes/fake"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/compute/kube"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy/generator"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/envoy/server"
	configFake "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
	policyFake "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned/fake"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/models"
	"github.com/openservicemesh/osm/pkg/signals"
	"github.com/openservicemesh/osm/pkg/tests"
)

func setupCP(numProxies int) (*ControlPlane[map[string][]types.Resource], error) {
	var cp *ControlPlane[map[string][]types.Resource]
	ctx := context.Background()
	stop := signals.RegisterExitHandlers()
	msgBroker := messaging.NewBroker(stop)
	certManager := tresorFake.NewFake(time.Minute)

	// monitor namespace
	namespace := tests.Namespace
	nsObj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   namespace,
			Labels: map[string]string{constants.OSMKubeResourceMonitorAnnotation: tests.MeshName},
		},
	}
	proxyUUID := uuid.New()
	labels := map[string]string{constants.EnvoyUniqueIDLabelName: proxyUUID.String()}

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
				EnableWASMStats: false,
			},
		},
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tests.BookstoreV1ServiceName,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "10.0.0.1",
			Ports: []corev1.ServicePort{{
				Name: "servicePort",
				TargetPort: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 8080,
				},
				Protocol: corev1.ProtocolTCP,
				Port:     8080,
			}},
			Selector: labels,
		},
	}

	objects := []runtime.Object{nsObj, svc}

	for i := 0; i < numProxies; i++ {
		cnPrefix := models.NewXDSCertCNPrefix(uuid.New(), models.KindSidecar, identity.New(fmt.Sprintf("svc-%d", i), tests.Namespace))
		cert, err := certManager.IssueCertificate(certificate.ForCommonNamePrefix(cnPrefix))
		if err != nil {
			return nil, err
		}
		certX509, err := certificate.DecodePEMCertificate(cert.GetCertificateChain())
		if err != nil {
			return nil, err
		}

		pod := tests.NewPodFixture(namespace, fmt.Sprintf("pod-%d-%s", i, proxyUUID), tests.BookstoreServiceAccountName, tests.PodLabels)
		pod.Labels[constants.EnvoyUniqueIDLabelName] = proxyUUID.String()
		objects = append(objects, pod)
		ctx := peer.NewContext(ctx, &peer.Peer{
			AuthInfo: tests.NewMockAuthInfo(certX509),
		})
		defer cp.ProxyConnected(ctx, int64(i))
	}

	kubeClient := k8sClientFake.NewSimpleClientset(objects...)
	configClient := configFake.NewSimpleClientset(&meshConfig)
	policyClient := policyFake.NewSimpleClientset()

	kubeController, err := k8s.NewClient(tests.OsmNamespace, tests.OsmMeshConfigName, msgBroker,
		k8s.WithKubeClient(kubeClient, tests.MeshName),
		k8s.WithConfigClient(configClient),
		k8s.WithPolicyClient(policyClient),
	)
	if err != nil {
		return nil, fmt.Errorf("Failed to create informer collection: %w", err)
	}
	kubeProvider := kube.NewClient(kubeController)

	mc := catalog.NewMeshCatalog(
		kubeProvider,
		certManager,
	)

	xdsGenerator := generator.NewEnvoyConfigGenerator(mc, certManager)
	xdsServer := server.NewADSServer()
	proxyRegistry := registry.NewProxyRegistry()
	cp = NewControlPlane[map[string][]types.Resource](xdsServer, xdsGenerator, mc, proxyRegistry, certManager, msgBroker)

	return cp, nil
}

// func BenchmarkUpdate100Proxies(b *testing.B) {
// 	tassert := assert.New(b)
// 	cp, err := setupCP(b.N)
// 	if err != nil {
// 		b.Fatal(err)
// 	}
// 	tassert.Equal(cp.proxyRegistry.GetConnectedProxyCount(), b.N)

// 	testDuration := 5 * time.Minute

// 	for i := 0; i < b.N; i++ {
// 		b.Run(fmt.Sprintf("Update%dProxies", b.N), func(b *testing.B) {
// 			b.StopTimer()
// 			b.ResetTimer()
// 			b.StartTimer()

// 			timer := time.NewTicker(time.Second / time.Duration(i))
// 			eventsSent :=
// 			for {
// 				select {
// 				case <-timer.C:
// 					// add event to queue, count num events
// 				}
// 			}

// 			// check ProxyBroadcastEventCount vs number actually sent.. that's the amount that is coalesced.
// 			// check time since last update
// 		})
// 	}
// }

// func BenchmarkSendXDSResponse(b *testing.B) {
// 	if err := logger.SetLogLevel("error"); err != nil {
// 		b.Logf("Failed to set log level to error: %s", err)
// 	}

// 	proxy, g := setupTestGenerator(b)
// 	for xdsType, generator := range g.generators {
// 		b.Run(string(xdsType), func(b *testing.B) {
// 			b.ResetTimer()
// 			b.StartTimer()
// 			for i := 0; i < b.N; i++ {
// 				if _, err := generator(context.Background(), proxy); err != nil {
// 					b.Fatalf("Failed to send response: %s", err)
// 				}
// 			}
// 		})
// 	}
// }
