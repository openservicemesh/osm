package smi

import (
	"flag"
	"fmt"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/version"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/tester"
	"github.com/deislabs/smc/pkg/tests"
	"github.com/deislabs/smi-sdk-go/pkg/apis/split/v1alpha2"
	"github.com/deislabs/smi-sdk-go/pkg/gen/client/split/clientset/versioned/fake"
)

var _ = Describe("Tests `appgw.ConfigBuilder`", func() {
	var k8sClient kubernetes.Interface
	var meshSpecClient *MeshSpec
	var smiClient *fake.Clientset
	var stopChannel chan struct{}

	version.Version = "a"
	version.GitCommit = "b"
	version.BuildDate = "c"

	ingressNS := "test-ingress-controller"
	serviceName := "hello-world"

	// Frontend and Backend port.
	servicePort := endpoint.Port(80)
	backendName := "http"
	backendPort := endpoint.Port(1356)

	// Endpoints
	endpoint1 := "1.1.1.1"
	endpoint2 := "1.1.1.2"
	endpoint3 := "1.1.1.3"

	// Create the "test-ingress-controller" namespace.
	// We will create all our resources under this namespace.
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ingressNS,
		},
	}

	// Create a node
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-1",
		},
		Spec: v1.NodeSpec{
			ProviderID: "azure:///subscriptions/subid/resourceGroups/MC_aksresgp_aksname_location/providers/Microsoft.Compute/virtualMachines/vmname",
		},
	}

	trafficSplit := v1alpha2.TrafficSplit{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "traffic-split",
			Namespace: "ns",
		},
		Spec: v1alpha2.TrafficSplitSpec{
			Service: "",
			Backends: []v1alpha2.TrafficSplitBackend{
				{
					Service: "service-one",
					Weight:  100,
				},
			},
		},
	}

	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: ingressNS,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name: "servicePort",
					TargetPort: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: backendName,
					},
					Protocol: v1.ProtocolTCP,
					Port:     int32(servicePort),
				},
			},
			Selector: map[string]string{"app": "frontend"},
		},
	}

	// Ideally we should be creating the `pods` resource instead of the `endpoints` resource
	// and allowing the k8s API server to create the `endpoints` resource which we end up consuming.
	// However since we are using a fake k8s client the resources are dumb which forces us to create the final
	// expected resource manually.
	endpoints := &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: ingressNS,
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP: endpoint1,
					},
					{
						IP: endpoint2,
					},
					{
						IP: endpoint3,
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "servicePort",
						Port:     int32(backendPort),
						Protocol: v1.ProtocolTCP,
					},
				},
			},
		},
	}

	pod := tests.NewPodFixture(serviceName, ingressNS, backendName, int32(backendPort))

	_ = flag.Lookup("logtostderr").Value.Set("true")
	_ = flag.Set("v", "3")

	testTrafficSplit := func() []*v1alpha2.TrafficSplit {
		// Get all the ingresses
		trafficSplits := (*meshSpecClient).ListTrafficSplits()
		// There should be only one ingress
		Expect(len(trafficSplits)).To(Equal(1), "Expected only one TrafficSplit resource but got: %d", len(trafficSplits))
		// Make sure it is the ingress we stored.
		Expect(trafficSplits[0]).To(Equal(trafficSplit))

		return trafficSplits
	}

	BeforeEach(func() {
		stopChannel = make(chan struct{})

		// Create the mock K8s client.
		k8sClient = testclient.NewSimpleClientset()
		smiClient = fake.NewSimpleClientset(&trafficSplit)
		Ω(smiClient).ToNot(BeNil())

		_, err := k8sClient.CoreV1().Namespaces().Create(ns)
		Ω(err).ToNot(HaveOccurred(), "Unable to create the namespace %s: %v", ingressNS, err)

		_, err = k8sClient.CoreV1().Nodes().Create(node)
		Ω(err).ToNot(HaveOccurred(), "Unable to create node resource due to: %v", err)

		// Load services.
		for _, obj := range tester.LoadService() {
			fmt.Printf("obj: %+v", obj)
			fmt.Print(service)
			// _, err = k8sClient.CoreV1().Services(ingressNS).Create(service)
			// Ω(err).ToNot(HaveOccurred(), "Unable to create service resource due to: %v", err)
		}

		// Create the endpoints associated with this service.
		_, err = k8sClient.CoreV1().Endpoints(ingressNS).Create(endpoints)
		Ω(err).ToNot(HaveOccurred(), "Unable to create endpoints resource due to: %v", err)

		// Create the pods associated with this service.
		_, err = k8sClient.CoreV1().Pods(ingressNS).Create(pod)
		Ω(err).ToNot(HaveOccurred(), "Unable to create pods resource due to: %v", err)

		// Create an SMI Client.
		stop := make(chan struct{})
		announcements := make(chan interface{})
		kubeConfig := rest.Config{}
		observeNamespaces := []string{"default"}
		kubeClient := kubernetes.NewForConfigOrDie(&kubeConfig)
		c := NewMeshSpecClient(kubeClient, &kubeConfig, observeNamespaces, announcements, stop)
		meshSpecClient = &c
		Expect(meshSpecClient).ShouldNot(BeNil(), "Unable to create `k8scontext`")
	})

	AfterEach(func() {
		close(stopChannel)
	})

	Context("Tests TrafficSplit", func() {
		It("Should be able to create Weighted Services from TrafficSplit CRD", func() {
			// Wait for the controller to receive an ingress update.
			// trafficSplitEvent()
			panic("XX")
			trafficSplits := testTrafficSplit()
			fmt.Printf("Here are the traffic splits: %+v", trafficSplits)
		})
	})
})
