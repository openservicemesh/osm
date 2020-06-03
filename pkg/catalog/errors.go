package catalog

import "errors"

var (
	errInvalidCertificateCN                  = errors.New("invalid cn")
	errMoreThanOnePodForCertificate          = errors.New("found more than one pod for certificate")
	errDidNotFindPodForCertificate           = errors.New("did not find pod for certificate")
	errNoServicesFoundForCertificate         = errors.New("no services found for certificate")
	errServiceAccountDoesNotMatchCertificate = errors.New("service account does not match certificate")
	errNamespaceDoesNotMatchCertificate      = errors.New("namespace does not match certificate")
	errServiceNotFoundForAnyProvider         = errors.New("no service found for service account with any of the mesh supported providers")
	errDomainNotFoundForService              = errors.New("no host/domain found to configure service")
	errNoTrafficSpecFoundForTrafficPolicy    = errors.New("no traffic spec found for the traffic policy")
)
