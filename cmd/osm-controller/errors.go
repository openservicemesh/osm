package main

import "github.com/pkg/errors"

var (
	errInvalidCertSecret = errors.New("Invalid secret for certificate")
)
