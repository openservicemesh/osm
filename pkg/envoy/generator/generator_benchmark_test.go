package generator

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sClientFake "k8s.io/client-go/kubernetes/fake"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/compute/kube"
	"github.com/openservicemesh/osm/pkg/k8s/informers"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/models"
	"github.com/openservicemesh/osm/pkg/signals"

	"github.com/openservicemesh/osm/pkg/certificate"

	"github.com/openservicemesh/osm/pkg/constants"
	configFake "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
	policyFake "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned/fake"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/tests"
)

func setupTestGenerator(b *testing.B) (*models.Proxy, *EnvoyConfigGenerator) {
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
	kubeController := k8s.NewClient(tests.OsmNamespace, tests.OsmMeshConfigName, informerCollection, kubeClient, policyClient, configClient, msgBroker)
	kubeProvider := kube.NewClient(kubeController)

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
	proxySvcAccount := tests.BookstoreServiceAccount

	proxyUUID := uuid.New()
	labels := map[string]string{constants.EnvoyUniqueIDLabelName: proxyUUID.String()}
	mc := catalog.NewMeshCatalog(
		kubeProvider,
		certManager,
		stop,
		msgBroker,
	)

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

	svc := tests.NewServiceFixture(tests.BookstoreV1ServiceName, namespace, labels)
	_, err = kubeClient.CoreV1().Services(namespace).Create(context.Background(), svc, metav1.CreateOptions{})
	if err != nil {
		b.Fatalf("Failed to create service: %v", err)
	}

	proxy := models.NewProxy(models.KindSidecar, proxyUUID, proxySvcAccount.ToServiceIdentity(), nil, 1)

	return proxy, NewEnvoyConfigGenerator(mc, certManager)
}

func BenchmarkSendXDSResponse(b *testing.B) {
	if err := logger.SetLogLevel("error"); err != nil {
		b.Logf("Failed to set log level to error: %s", err)
	}

	proxy, g := setupTestGenerator(b)
	for xdsType, generator := range g.generators {
		b.Run(string(xdsType), func(b *testing.B) {
			b.ResetTimer()
			b.StartTimer()
			for i := 0; i < b.N; i++ {
				if _, err := generator(context.Background(), proxy); err != nil {
					b.Fatalf("Failed to send response: %s", err)
				}
			}
		})
	}
}
