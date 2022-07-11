package injector

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	mapset "github.com/deckarep/golang-set"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	configFake "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
	policyFake "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned/fake"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/k8s/informers"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/signals"
	"github.com/openservicemesh/osm/pkg/tests"
)

func BenchmarkPodCreationHandler(b *testing.B) {
	if err := logger.SetLogLevel("error"); err != nil {
		b.Logf("Failed to set log level to error: %s", err)
	}

	admissionRequestBody := `{
		"kind": "AdmissionReview",
		"apiVersion": "admission.k8s.io/v1",
		"request": {
		  "uid": "11111111-2222-3333-4444-555555555555",
		  "kind": {
			"group": "",
			"version": "v1",
			"kind": "PodExecOptions"
		  },
		  "resource": {
			"group": "",
			"version": "v1",
			"resource": "pods"
		  },
		  "subResource": "exec",
		  "requestKind": {
			"group": "",
			"version": "v1",
			"kind": "PodExecOptions"
		  },
		  "requestResource": {
			"group": "",
			"version": "v1",
			"resource": "pods"
		  },
		  "requestSubResource": "exec",
		  "name": "some-pod-1111111111-22222",
		  "namespace": "default",
		  "operation": "CONNECT",
		  "userInfo": {
			"username": "user",
			"groups": []
		  },
		  "object": {
			"kind": "PodExecOptions",
			"apiVersion": "v1",
			"stdin": true,
			"stdout": true,
			"tty": true,
			"container": "some-pod",
			"command": ["bin/bash"]
		  },
		  "oldObject": null,
		  "dryRun": false,
		  "options": null
		}
	  }`

	kubeClient := testclient.NewSimpleClientset()
	configClient := configFake.NewSimpleClientset()
	policyClient := policyFake.NewSimpleClientset()
	stop := signals.RegisterExitHandlers()
	msgBroker := messaging.NewBroker(stop)
	informerCollection, err := informers.NewInformerCollection(tests.MeshName, stop,
		informers.WithKubeClient(kubeClient),
		informers.WithConfigClient(configClient, tests.OsmMeshConfigName, tests.OsmNamespace),
	)
	kubeController := k8s.NewKubernetesController(informerCollection, policyClient, msgBroker)
	if err != nil {
		b.Fatalf("Failed to create kubeController: %s", err.Error())
	}

	testNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
	}
	if _, err := kubeClient.CoreV1().Namespaces().Create(context.TODO(), testNamespace, metav1.CreateOptions{}); err != nil {
		b.Fatalf("Failed to create namespace: %s", err.Error())
	}

	wh := &mutatingWebhook{
		kubeClient:          kubeClient,
		kubeController:      kubeController,
		nonInjectNamespaces: mapset.NewSet(),
	}

	b.ResetTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/a/b/c", strings.NewReader(admissionRequestBody))
		req.Header = map[string][]string{
			"Content-Type": {"application/json"},
		}

		wh.podCreationHandler(w, req)
		res := w.Result()
		if res.StatusCode != 200 {
			b.Fatalf("Expected 200, got %d", res.StatusCode)
		}
	}
	b.StopTimer()
}
