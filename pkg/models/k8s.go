package models

// Secret represents secret for k8s abstraction
type Secret struct {
	Name      string
	Namespace string
	Labels    map[string]string
	Data      map[string][]byte
}
