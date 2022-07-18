package injector

import "fmt"

var (
	errNamespaceNotFound   = fmt.Errorf("namespace not found")
	errNilAdmissionRequest = fmt.Errorf("nil admission request")
)
