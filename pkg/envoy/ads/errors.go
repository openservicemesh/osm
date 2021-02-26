package ads

import (
	"github.com/pkg/errors"
)

var errUnknownTypeURL = errors.New("unknown TypeUrl")
var errGrpcClosed = errors.New("grpc closed")
