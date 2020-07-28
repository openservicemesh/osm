package httpserver

import (
	"net/http"

	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("http-server")
)

// HTTPServer serving probes and metrics
type HTTPServer interface {
	Start()
	Stop()
}

type httpServer struct {
	server *http.Server
}
