package k8s

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
)

// FakeDiscoveryClient is a fake client for k8s API discovery
type FakeDiscoveryClient struct {
	discovery.ServerResourcesInterface
	Resources map[string]metav1.APIResourceList
	Err       error
}

// ServerResourcesForGroupVersion returns the supported resources for a group and version.
func (f *FakeDiscoveryClient) ServerResourcesForGroupVersion(groupVersion string) (*metav1.APIResourceList, error) {
	resp := f.Resources[groupVersion]
	return &resp, f.Err
}
