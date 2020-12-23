package ads

import (
	"github.com/pkg/errors"
)

var errUnknownTypeURL = errors.New("unknown TypeUrl")
var errCreatingResponse = errors.New("creating response")
var errGrpcClosed = errors.New("grpc closed")
