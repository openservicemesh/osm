package webhook

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

// CertRotatedFunc is a callback to perform other actions when the server's HTTPS cert gets rotated.
type CertRotatedFunc func(cert *certificate.Certificate) error

// Server is a construct to run generic HTTPS webhook servers.
type Server struct {
	name         string
	namespace    string
	cm           *certificate.Manager
	server       *http.Server
	onCertChange CertRotatedFunc

	mu   sync.Mutex
	cert tls.Certificate
}

// NewServer returns a new server based on the input. Run() must be called to start the server.
func NewServer(name, namespace string, port int, cm *certificate.Manager, handlers map[string]http.HandlerFunc, onCertChange CertRotatedFunc) *Server {
	mux := http.NewServeMux()

	for path, h := range handlers {
		mux.Handle(path, metricsstore.AddHTTPMetrics(h))
	}

	s := &Server{
		name:         name,
		namespace:    namespace,
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
			MinVersion: constants.MinTLSVersion,
		},
	}
	return s
}

// Run actually starts the server.
func (s *Server) Run(ctx context.Context) error {
	if err := s.configureCertificateRotation(ctx); err != nil {
		return err
	}

	log.Info().Msgf("Starting %s webhook server on: %s", s.name, s.server.Addr)
	go func() {
		err := s.server.ListenAndServeTLS("", "") // err is always non-nil
		log.Error().Err(err).Msgf("%s webhook HTTP server shutdown", s.name)
	}()

	go func() {
		// Wait on exit signals
		<-ctx.Done()

		// Stop the servers
		if err := s.server.Shutdown(context.TODO()); err != nil {
			log.Error().Err(err).Msgf("Error shutting down %s webhook HTTP server", s.name)
		} else {
			log.Info().Msgf("Done shutting down %s webhook HTTP server", s.name)
		}
	}()
	return nil
}

func (s *Server) setCert() error {
	// This is a certificate issued for the webhook handler
	// This cert does not have to be related to the Envoy certs, but it does have to match
	// the cert provisioned.
	// Kubernetes requires webhooks to have format of 'servicename.namespace.svc' without trust domain
	webhookCert, err := s.cm.IssueCertificate(certificate.ForCommonName(s.certCommonName()))
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

func (s *Server) certCommonName() string {
	return fmt.Sprintf("%s.%s.svc", s.name, s.namespace)
}

// configureCertificateRotation gets the certificate from the certificate manager and spawns a goroutine to watch for certificate rotation.
func (s *Server) configureCertificateRotation(ctx context.Context) error {
	// listen for certificate rotation first, so we don't miss any events
	certRotationChan, unsubscribeRotation := s.cm.SubscribeRotations(s.certCommonName())

	if err := s.setCert(); err != nil {
		// this is a fatal error on start, we can't continue without a cert
		unsubscribeRotation()
		return err
	}

	// Handle the rotations until the context is cancelled
	go func() {
		log.Info().Str("webhook", s.name).Str("cn", s.certCommonName()).Msg("Listening for certificate rotations")
		defer unsubscribeRotation()
		for {
			select {
			case <-certRotationChan:
				log.Debug().Str("webhook", s.name).Str("cn", s.certCommonName()).Msg("Certificate rotation was initiated for webhook")
				if err := s.setCert(); err != nil {
					events.GenericEventRecorder().ErrorEvent(err, events.CertificateRotationFailure, "Error rotating the certificate for webhook server")
					continue
				}
				log.Info().Str("webhook", s.name).Str("cn", s.certCommonName()).Msg("Certificate rotated for webhook")
			case <-ctx.Done():
				log.Info().Str("webhook", s.name).Str("cn", s.certCommonName()).Msg("Stop listening for certificate rotations")
				return
			}
		}
	}()
	return nil
}
