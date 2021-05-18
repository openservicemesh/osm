// Command osm-webhook starts up a Kubernetes Validating Webhook on the
// specified port, listening for requests over HTTPS.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/health"
	"github.com/openservicemesh/osm/pkg/httpserver"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/signals"
	"github.com/openservicemesh/osm/pkg/version"
	"github.com/spf13/pflag"
)

var (
	verbosity string
)

var (
	flags = pflag.NewFlagSet(`osm-webhook`, pflag.ExitOnError)
	port  = flags.Int("port", constants.OSMWebhookPort, "osm webhook port")
	log   = logger.New("osm-webhook/main")
)

func init() {
	flags.StringVarP(&verbosity, "verbosity", "v", "info", "Set log verbosity level")

}

func parseFlags() error {
	if err := flags.Parse(os.Args); err != nil {
		return err
	}
	_ = flag.CommandLine.Parse([]string{})
	return nil
}

func main() {
	log.Info().Msgf("Starting osm-webhook %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)

	if err := logger.SetLogLevel(verbosity); err != nil {
		log.Fatal().Err(err).Msg("Error setting log level")
	}

	stop := signals.RegisterExitHandlers()

	httpServer := httpserver.NewHTTPServer(uint16(*port))

	httpServer.AddHandler("/version", version.GetVersionHandler())

	httpServer.AddHandler("/", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprint(w, "hello world")
	}))

	// TODO: add health/readiness probes
	httpServer.AddHandlers(map[string]http.Handler{
		"/health/ready": health.ReadinessHandler(nil, nil),
		"/health/alive": health.LivenessHandler(nil, nil),
	})

	// TODO: Do we need to add metrics stuff?

	// TODO: Add SSL Certs

	err := httpServer.Start()
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to start OSM metrics/probes HTTP server")
	}

	<-stop
	log.Info().Msgf("Stopping osm-webhook %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)

}
