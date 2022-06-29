// Package health implements functionality for readiness and liveness health probes. The probes are served
// by an HTTP server that exposes HTTP paths to probe on, with this package providing the necessary HTTP
// handlers to respond to probe requests.
package health

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/logger"
)

var log = logger.New("health")

// Probes is the interface for liveness and readiness probes
type Probes interface {
	Liveness() bool
	Readiness() bool
	GetID() string
}

// ProtocolType identifies the protocol used for a connection
type ProtocolType string

const (
	// ProtocolHTTP means that the protocol used will be http://
	ProtocolHTTP ProtocolType = "http"

	// ProtocolHTTPS means that the protocol used will be https://
	ProtocolHTTPS ProtocolType = "https"
)

// HTTPProbe is a type used to represent an HTTP or HTTPS probe
type HTTPProbe struct {
	URL      string
	Protocol ProtocolType
}

// Probe sends an HTTP or HTTPS probe for the given probe request.
// Certificate verification is skipped for HTTPS probes.
func (httpProbe HTTPProbe) Probe() (int, error) {
	client := &http.Client{}

	if httpProbe.Protocol == ProtocolHTTPS {
		// Certificate validation is to be skipped for HTTPS probes
		// similar to how k8s api server handles HTTPS probes.
		// #nosec G402
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				MinVersion:         tls.VersionTLS12,
			},
		}
		client.Transport = transport
	}

	req, err := http.NewRequest("GET", httpProbe.URL, nil)
	if err != nil {
		return http.StatusServiceUnavailable, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return http.StatusServiceUnavailable, err
	}

	//nolint: errcheck
	//#nosec G307
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

func setProbeResponse(w http.ResponseWriter, responseCode int, msg string) {
	w.WriteHeader(responseCode)
	_, _ = w.Write([]byte(msg))
}

// ReadinessHandler returns readiness http handlers for health
func ReadinessHandler(probes []Probes, urlProbes []HTTPProbe) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Probe on all configured probes
		for _, probe := range probes {
			if !probe.Readiness() {
				msg := fmt.Sprintf("Readiness probe for %s indicates it is not ready", probe.GetID())
				log.Warn().Msgf(msg)
				setProbeResponse(w, http.StatusServiceUnavailable, msg)
				return
			}
		}

		// Probe on all configured URLs
		for _, urlProbe := range urlProbes {
			responseCode, err := urlProbe.Probe()
			if err != nil || responseCode != http.StatusOK {
				msg := fmt.Sprintf("Readiness probe failed for URL %s: %s", urlProbe.URL, err)
				log.Warn().Msgf(msg)
				setProbeResponse(w, responseCode, msg)
				return
			}
		}

		setProbeResponse(w, http.StatusOK, constants.ServiceReadyResponse)
	})
}

// LivenessHandler returns liveness http handlers for health
func LivenessHandler(probes []Probes, urlProbes []HTTPProbe) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Probe on all configured probes
		for _, probe := range probes {
			if !probe.Liveness() {
				msg := fmt.Sprintf("Liveness probe for %s indicates it is not alive", probe.GetID())
				log.Warn().Msgf(msg)
				setProbeResponse(w, http.StatusServiceUnavailable, msg)
				return
			}
		}

		// Probe on all configured URLs
		for _, urlProbe := range urlProbes {
			responseCode, err := urlProbe.Probe()
			if err != nil || responseCode != http.StatusOK {
				msg := fmt.Sprintf("Liveness probe failed for URL %s: %s", urlProbe.URL, err)
				log.Warn().Msgf(msg)
				setProbeResponse(w, responseCode, msg)
				return
			}
		}

		setProbeResponse(w, http.StatusOK, constants.ServiceAliveResponse)
	})
}

// SimpleHandler returns a simple http handler for health checks.
func SimpleHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("Health OK")); err != nil {
		log.Error().Err(err).Msg("Error writing bytes for crd-conversion webhook health check handler")
	}
}
