package webhook

import "fmt"

var (
	errEmptyAdmissionRequestBody = fmt.Errorf("empty request admission request body")
)
