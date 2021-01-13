package catalog

import "github.com/pkg/errors"

var (
	errInvalidCertificateCN                  = errors.New("invalid cn")
	errMoreThanOnePodForCertificate          = errors.New("found more than one pod for certificate")
	errDidNotFindPodForCertificate           = errors.New("did not find pod for certificate")
	errServiceAccountDoesNotMatchCertificate = errors.New("service account does not match certificate")
	errNamespaceDoesNotMatchCertificate      = errors.New("namespace does not match certificate")
	errServiceNotFoundForAnyProvider         = errors.New("no service found for service account with any of the mesh supported providers")
	errNoTrafficSpecFoundForTrafficPolicy    = errors.New("no traffic spec found for the traffic policy")
	errServiceNotFound                       = errors.New("service not found")
)
