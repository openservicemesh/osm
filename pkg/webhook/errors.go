package webhook

import "github.com/pkg/errors"

var (
	errEmptyAdmissionRequestBody = errors.New("empty request admission request body")
)
