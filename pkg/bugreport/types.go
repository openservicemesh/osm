// Package bugreport implements functionality related to generating bug reports.
package bugreport

import (
	"io"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// Config is the type used to hold the bug report configuration
type Config struct {
	Stdout                  io.Writer
	Stderr                  io.Writer
	KubeClient              kubernetes.Interface
	ControlPlaneNamepace    string
	AppNamespaces           []string
	AppDeployments          []types.NamespacedName
	AppPods                 []types.NamespacedName
	OutFile                 string
	stagingDir              string
	CollectControlPlaneLogs bool
}
