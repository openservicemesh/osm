package webhook

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

// CertRotatedFunc is a callback to perform other actions when the server's HTTPS cert gets rotated.
type CertRotatedFunc func(cert *certificate.Certificate) error

// Server is a construct to run generic HTTPS webhook servers.
type Server struct {
	name      string
	namespace string
	cm        *certificate.Manager
	broker    *messaging.Broker

	server       *http.Server
	onCertChange CertRotatedFunc

	mu   sync.Mutex
	cert tls.Certificate
}

// NewServer returns a new server based on the input. Run() must be called to start the server.
func NewServer(name, namespace string, port int, cm *certificate.Manager, broker *messaging.Broker, handlers map[string]http.HandlerFunc, onCertChange CertRotatedFunc) (*Server, error) {
	mux := http.NewServeMux()

	for path, h := range handlers {
		mux.Handle(path, metricsstore.AddHTTPMetrics(h))
	}

	s := &Server{
		name:         name,
		namespace:    namespace,
		broker:       broker,
		cm:           cm,
		onCertChange: onCertChange,
	}
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
		// #nosec G402
		TLSConfig: &tls.Config{
			GetCertificate: func(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
				s.mu.Lock()
				defer s.mu.Unlock()
				return &s.cert, nil
			},
			MinVersion: tls.VersionTLS13,
		},
	}

	// This is a certificate issued for the webhook handler
	// This cert does not have to be related to the Envoy certs, but it does have to match
	// the cert provisioned with the ConversionWebhook on the CRD's
	webhookCert, err := s.cm.IssueCertificate(
		s.certName(),
		certificate.WithValidityPeriod(constants.XDSCertificateValidityPeriod),
		certificate.FullCNProvided())
	if err != nil {
		return nil, err
	}

	if err := s.setCert(webhookCert); err != nil {
		return nil, err
	}
	return s, nil
}

// Run actually starts the server. It blocks until the passed in context is done.
func (s *Server) Run(ctx context.Context) {
	log.Info().Msgf("Starting conversion webhook server on: %s", s.server.Addr)
	go s.watchForCertChange(ctx)
	go func() {
		err := s.server.ListenAndServeTLS("", "") // err is always non-nil
		log.Error().Err(err).Msg("crd-converter webhook HTTP server failed to start")
	}()

	// Wait on exit signals
	<-ctx.Done()

	// Stop the servers
	if err := s.server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Error shutting down crd-conversion webhook HTTP server")
	} else {
		log.Info().Msg("Done shutting down crd-conversion webhook HTTP server")
	}
}

func (s *Server) watchForCertChange(ctx context.Context) {
	// NOTE:
	// 1. We could optimize this by filtering to a topic. We could do this by adding an option to the certmanager
	// to publish events on a topic that matches the CN.

	// 2. Regardless of filtering, due to the semantics of pubsub (#4847), whenever this goroutine blocks (ie: making a
	// call to onCertChange), the enrite Cert Pub Sub will block, preventing all publishes to that chan. That can
	// further block up those goroutines, ie: preventing certificates from being rotated.
	ch := s.broker.GetCertPubSub().Sub(string(announcements.CertificateRotated))
	for {
		select {
		case event := <-ch:
			cert, ok := event.(events.PubSubMessage).NewObj.(*certificate.Certificate)
			if ok && cert.GetCommonName().String() == s.certName() {
				// TODO(4619) implement retries
				if err := s.onCertChange(cert); err != nil {
					log.Err(err).Msgf("error acting on certificate rotation for %s", s.name)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Server) certName() string {
	return fmt.Sprintf("%s.%s.svc", s.name, s.namespace)
}

func (s *Server) setCert(webhookCert *certificate.Certificate) error {
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
