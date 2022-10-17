package models

// Secret represents secret for k8s abstraction
type Secret struct {
	Name      string
	Namespace string
	Data      map[string][]byte
}
