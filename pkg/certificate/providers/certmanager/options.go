package certmanager

import (
	"errors"

	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
)

// ValidateCertManagerOptions validates the options for cert-manager.io certificate provider
func (options *Options) Validate() error {
	if options.IssuerName == "" {
		return errors.New("IssuerName not specified in cert-manager.io options")
	}

	if options.IssuerKind == "" {
		return errors.New("IssuerKind not specified in cert-manager.io options")
	}

	if options.IssuerGroup == "" {
		return errors.New("IssuerGroup not specified in cert-manager.io options")
	}

	return nil
}

func (options *Options) IssuerRef() cmmeta.ObjectReference {
	return cmmeta.ObjectReference{
		Name:  options.IssuerName,
		Kind:  options.IssuerKind,
		Group: options.IssuerGroup,
	}
}
