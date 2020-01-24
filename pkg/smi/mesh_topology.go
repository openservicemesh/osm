package smi

import (
	"fmt"

	"github.com/deislabs/smc/pkg/endpoint"

	TrafficTarget "github.com/deislabs/smi-sdk-go/pkg/apis/access/v1alpha1"
	TrafficSpec "github.com/deislabs/smi-sdk-go/pkg/apis/specs/v1alpha1"
	"github.com/deislabs/smi-sdk-go/pkg/apis/split/v1alpha2"
	smiTrafficTargetClient "github.com/deislabs/smi-sdk-go/pkg/gen/client/access/clientset/versioned"
	smiTrafficSpecClient "github.com/deislabs/smi-sdk-go/pkg/gen/client/specs/clientset/versioned"
	smiTrafficSplitClient "github.com/deislabs/smi-sdk-go/pkg/gen/client/split/clientset/versioned"
	"github.com/eapache/channels"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/deislabs/smc/pkg/mesh"
)

// We have a few different k8s clients. This identifies these in logs.
const kubernetesClientName = "MeshTopology"

// NewMeshTopologyClient creates the Kubernetes client, which retrieves SMI specific CRDs.
func NewMeshTopologyClient(kubeConfig *rest.Config, namespaces []string, announceChan *channels.RingChannel, stopChan chan struct{}) mesh.Topology {
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	smiTrafficSplitClientSet := smiTrafficSplitClient.NewForConfigOrDie(kubeConfig)
	smiTrafficSpecClientSet := smiTrafficSpecClient.NewForConfigOrDie(kubeConfig)
	smiTrafficTargetClientSet := smiTrafficTargetClient.NewForConfigOrDie(kubeConfig)
	client := newSMIClient(kubeClient, smiTrafficSplitClientSet, smiTrafficSpecClientSet, smiTrafficTargetClientSet, namespaces, announceChan, kubernetesClientName)
	err := client.Run(stopChan)
	if err != nil {
		glog.Fatalf("Could not start %s client: %s", kubernetesClientName, err)
	}
	return client
}

// ListTrafficSplits implements mesh.Topology by returning the list of traffic splits.
func (c *Client) ListTrafficSplits() []*v1alpha2.TrafficSplit {
	var trafficSplits []*v1alpha2.TrafficSplit
	for _, splitIface := range c.caches.TrafficSplit.List() {
		split := splitIface.(*v1alpha2.TrafficSplit)
		trafficSplits = append(trafficSplits, split)
	}
	return trafficSplits
}

// ListServices implements mesh.Topology by returning the services observed from the given compute provider
func (c *Client) ListServices() []endpoint.ServiceName {
	// TODO(draychev): split the namespace and the service kubernetesClientName -- for non-kubernetes services we won't have namespace
	var services []endpoint.ServiceName
	for _, splitIface := range c.caches.TrafficSplit.List() {
		split := splitIface.(*v1alpha2.TrafficSplit)
		namespacedServiceName := fmt.Sprintf("%s/%s", split.Namespace, split.Spec.Service)
		services = append(services, endpoint.ServiceName(namespacedServiceName))
		for _, backend := range split.Spec.Backends {
			namespacedServiceName := fmt.Sprintf("%s/%s", split.Namespace, backend.Service)
			services = append(services, endpoint.ServiceName(namespacedServiceName))
		}
	}
	return services
}

// GetService retrieves the Kubernetes Services resource for the given ServiceName.
func (c *Client) GetService(svc endpoint.ServiceName) (service *v1.Service, exists bool, err error) {
	svcIf, exists, err := c.caches.Services.GetByKey(string(svc))
	if exists && err == nil {
		return svcIf.(*v1.Service), exists, err
	}
	return nil, exists, err
}

// ListHTTPTrafficSpecs implements mesh.Topology by returning the list of traffic specs.
func (c *Client) ListHTTPTrafficSpecs() []*TrafficSpec.HTTPRouteGroup {
	var httpTrafficSpec []*TrafficSpec.HTTPRouteGroup
	for _, specIface := range c.caches.TrafficSpec.List() {
		spec := specIface.(*TrafficSpec.HTTPRouteGroup)
		httpTrafficSpec = append(httpTrafficSpec, spec)
	}
	return httpTrafficSpec
}

// ListTrafficTargets implements mesh.Topology by returning the list of traffic targets.
func (c *Client) ListTrafficTargets() []*TrafficTarget.TrafficTarget {
	var trafficTarget []*TrafficTarget.TrafficTarget
	for _, targetIface := range c.caches.TrafficTarget.List() {
		target := targetIface.(*TrafficTarget.TrafficTarget)
		trafficTarget = append(trafficTarget, target)
	}
	return trafficTarget
}
