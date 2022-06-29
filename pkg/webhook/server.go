package webhook

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

// CertRotatedFunc is a callback to perform other actions when the server's HTTPS cert gets rotated.
type CertRotatedFunc func(cert *certificate.Certificate) error

// Server is a construct to run generic HTTPS webhook servers.
type Server struct {
	name         string
	cm           *certificate.Manager
	server       *http.Server
	onCertChange CertRotatedFunc

	mu   sync.Mutex
	cert tls.Certificate
}

// NewServer returns a new server based on the input. Run() must be called to start the server.
func NewServer(name, namespace string, port int, cm *certificate.Manager, handlers map[string]http.HandlerFunc, onCertChange CertRotatedFunc) (*Server, error) {
	mux := http.NewServeMux()

	for path, h := range handlers {
		mux.Handle(path, metricsstore.AddHTTPMetrics(h))
	}

	s := &Server{
		name:         name,
		cm:           cm,
		onCertChange: onCertChange,
	}
	s.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: time.Second * 10,
		// #nosec G402
		TLSConfig: &tls.Config{
			GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
				s.mu.Lock()
				defer s.mu.Unlock()
				return &s.cert, nil
			},
			MinVersion: tls.VersionTLS12,
		},
	}
	// set the certificate once, which will also call onCertChange.
	if err := s.setCert(name, namespace); err != nil {
		return nil, err
	}
	return s, nil
}

// Run actually starts the server. It blocks until the passed in context is done.
func (s *Server) Run(ctx context.Context) {
	log.Info().Msgf("Starting %s webhook server on: %s", s.name, s.server.Addr)
	go func() {
		err := s.server.ListenAndServeTLS("", "") // err is always non-nil
		log.Error().Err(err).Msgf("%s webhook HTTP server shutdown", s.name)
	}()

	// Wait on exit signals
	<-ctx.Done()

	// Stop the servers
	if err := s.server.Shutdown(context.TODO()); err != nil {
		log.Error().Err(err).Msgf("Error shutting down %s webhook HTTP server", s.name)
	} else {
		log.Info().Msgf("Done shutting down %s webhook HTTP server", s.name)
	}
}

func (s *Server) setCert(name, namespace string) error {
	// This is a certificate issued for the webhook handler
	// This cert does not have to be related to the Envoy certs, but it does have to match
	// the cert provisioned.
	webhookCert, err := s.cm.IssueCertificate(
		fmt.Sprintf("%s.%s.svc", name, namespace),
		certificate.Internal,
		certificate.FullCNProvided())
	if err != nil {
		return err
	}

	// Generate a key pair from your pem-encoded cert and key ([]byte).
	cert, err := tls.X509KeyPair(webhookCert.GetCertificateChain(), webhookCert.GetPrivateKey())
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.cert = cert
	s.mu.Unlock()

	return s.onCertChange(webhookCert)
}
