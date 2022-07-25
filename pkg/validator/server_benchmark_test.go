package validator

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	k8sClientFake "k8s.io/client-go/kubernetes/fake"

	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiAccessClientFake "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/clientset/versioned/fake"
	smiSpecClientFake "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/specs/clientset/versioned/fake"
	smiSplitClientFake "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/clientset/versioned/fake"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	configFake "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
	policyFake "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned/fake"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/k8s/informers"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/policy"
	"github.com/openservicemesh/osm/pkg/signals"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/webhook"
)

func BenchmarkDoValidation(b *testing.B) {
	if err := logger.SetLogLevel("error"); err != nil {
		b.Logf("Failed to set log level to error: %s", err)
	}

	kubeClient := k8sClientFake.NewSimpleClientset()
	_, cancel := context.WithCancel(context.Background())
	stop := signals.RegisterExitHandlers(cancel)
	msgBroker := messaging.NewBroker(stop)
	smiTrafficSplitClientSet := smiSplitClientFake.NewSimpleClientset()
	smiTrafficSpecClientSet := smiSpecClientFake.NewSimpleClientset()
	smiTrafficTargetClientSet := smiAccessClientFake.NewSimpleClientset()
	policyClient := policyFake.NewSimpleClientset()
	configClient := configFake.NewSimpleClientset()
	informerCollection, err := informers.NewInformerCollection(tests.MeshName, stop,
		informers.WithKubeClient(kubeClient),
		informers.WithSMIClients(smiTrafficSplitClientSet, smiTrafficSpecClientSet, smiTrafficTargetClientSet),
		informers.WithConfigClient(configClient, tests.OsmMeshConfigName, tests.OsmNamespace),
		informers.WithPolicyClient(policyClient),
	)
	if err != nil {
		b.Fatalf("Failed to create informer collection: %s", err)
	}
	k8sClient := k8s.NewKubernetesController(informerCollection, policyClient, msgBroker)
	policyController := policy.NewPolicyController(informerCollection, k8sClient, msgBroker)
	kv := &policyValidator{
		policyClient: policyController,
	}

	w := httptest.NewRecorder()
	s := &validatingWebhookServer{
		validators: map[string]validateFunc{
			policyv1alpha1.SchemeGroupVersion.WithKind("IngressBackend").String():         kv.ingressBackendValidator,
			policyv1alpha1.SchemeGroupVersion.WithKind("Egress").String():                 egressValidator,
			policyv1alpha1.SchemeGroupVersion.WithKind("UpstreamTrafficSetting").String(): kv.upstreamTrafficSettingValidator,
			smiAccess.SchemeGroupVersion.WithKind("TrafficTarget").String():               trafficTargetValidator,
		},
	}

	b.ResetTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		req := &http.Request{
			Header: map[string][]string{
				webhook.HTTPHeaderContentType: {webhook.ContentTypeJSON},
			},
			Body: io.NopCloser(strings.NewReader(`{
				"metadata": {
					"uid": "some-uid"
				},
				"request": {}
			}`)),
		}
		s.doValidation(w, req)
		res := w.Result()
		if res.StatusCode != http.StatusOK {
			b.Fatalf("Expected status code %d, got %d", http.StatusOK, res.StatusCode)
		}
	}
	b.StopTimer()
}
