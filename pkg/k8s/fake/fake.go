package fake

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
)

// DiscoveryClient is a fake client for k8s API discovery
type DiscoveryClient struct {
	discovery.ServerResourcesInterface
	Resources map[string]metav1.APIResourceList
	Err       error
}

// ServerResourcesForGroupVersion returns the supported resources for a group and version.
func (f *DiscoveryClient) ServerResourcesForGroupVersion(groupVersion string) (*metav1.APIResourceList, error) {
	resp := f.Resources[groupVersion]
	return &resp, f.Err
}
