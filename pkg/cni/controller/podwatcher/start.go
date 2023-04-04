package podwatcher

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
)

// Run start to run controller to watch
func Run(client kubernetes.Interface, stop chan struct{}) error {
	// run local ip controller
	if err := runLocalPodController(client, stop); err != nil {
		return fmt.Errorf("run local ip controller error: %v", err)
	}
	return nil
}
