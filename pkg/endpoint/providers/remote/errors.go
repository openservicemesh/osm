package remote

import "github.com/pkg/errors"

var (
	errSyncingCaches                      = errors.New("failed initial sync of resources required for ingress")
	errInitInformers                      = errors.New("informers are not initialized")
	errDidNotFindServiceForServiceAccount = errors.New("no service exists for the service account")
	errMoreThanServiceForServiceAccount   = errors.New("more than one service found for the service account")
	errInvalidSvcAccount                  = errors.New("invalid service account name")
	errServiceNotFound                    = errors.New("Service not found")
)
