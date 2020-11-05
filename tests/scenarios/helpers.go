package scenarios

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/tests"
)

func makeService(kubeClient kubernetes.Interface, svcName string) {
	// Create a service for the pod created above
	selectors := map[string]string{
		// These need to match the POD created above
		tests.SelectorKey: tests.SelectorValue,
	}
	// The serviceName must match the SMI
	service := tests.NewServiceFixture(svcName, tests.Namespace, selectors)
	_, err := kubeClient.CoreV1().Services(tests.Namespace).Create(context.TODO(), service, metav1.CreateOptions{})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
}

func makePod(kubeClient kubernetes.Interface, namespace, podName, serviceAccountName, proxyUUID string) (*v1.Pod, error) {
	// Create a pod to match the CN
	requestedPod := tests.NewPodTestFixtureWithOptions(namespace, podName, serviceAccountName)
	requestedPod.Labels[constants.EnvoyUniqueIDLabelName] = proxyUUID // This is what links the Pod and the Certificate
	createdPod, err := kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), &requestedPod, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return createdPod, nil
}

func getMeshCatalogAndProxy() (catalog.MeshCataloger, *envoy.Proxy, error) {
	kubeClient := testclient.NewSimpleClientset()

	meshCatalog := catalog.NewFakeMeshCatalog(kubeClient)

	if _, err := makePod(kubeClient, tests.Namespace, "bookbuyer", tests.BookbuyerServiceAccountName, tests.ProxyUUID); err != nil {
		return nil, nil, err
	}
	if _, err := makePod(kubeClient, tests.Namespace, "bookstore", tests.BookstoreServiceAccountName, uuid.New().String()); err != nil {
		return nil, nil, err
	}

	for _, svcName := range []string{tests.BookbuyerServiceName, tests.BookstoreApexServiceName, tests.BookstoreV1ServiceName, tests.BookstoreV2ServiceName} {
		makeService(kubeClient, svcName)
	}

	proxy := envoy.NewProxy(certificate.CommonName(fmt.Sprintf("%s.%s.%s", tests.ProxyUUID, tests.BookbuyerServiceAccountName, tests.Namespace)), nil)

	return meshCatalog, proxy, nil
}
