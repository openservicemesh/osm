package httpserver

import (
	"net/http"

	"github.com/open-service-mesh/osm/pkg/logger"
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
