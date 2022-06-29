package utils

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/openservicemesh/osm/pkg/certificate"
)

func setupMutualTLS(insecure bool, serverName string, certPem []byte, keyPem []byte, ca []byte) (grpc.ServerOption, error) {
	certif, err := tls.X509KeyPair(certPem, keyPem)
	if err != nil {
		return nil, fmt.Errorf("[grpc][mTLS][%s] Failed loading Certificate (%+v) and Key (%+v) PEM files", serverName, certPem, keyPem)
	}

	certPool := x509.NewCertPool()

	// Load the set of Root CAs
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		return nil, fmt.Errorf("[grpc][mTLS][%s] Failed to append client certs", serverName)
	}

	// #nosec G402
	tlsConfig := tls.Config{
		InsecureSkipVerify: insecure,
		ServerName:         serverName,
		ClientAuth:         tls.RequireAndVerifyClientCert,
		Certificates:       []tls.Certificate{certif},
		ClientCAs:          certPool,
		MinVersion:         tls.VersionTLS12,
	}
	return grpc.Creds(credentials.NewTLS(&tlsConfig)), nil
}

// ValidateClient ensures that the connected client is authorized to connect to the gRPC server.
func ValidateClient(ctx context.Context) (certificate.CommonName, certificate.SerialNumber, error) {
	mtlsPeer, ok := peer.FromContext(ctx)
	if !ok {
		log.Error().Msg("[grpc][mTLS] No peer found")
		return "", "", status.Error(codes.Unauthenticated, "no peer found")
	}

	tlsAuth, ok := mtlsPeer.AuthInfo.(credentials.TLSInfo)
	if !ok {
		log.Error().Msg("[grpc][mTLS] Unexpected peer transport credentials")
		return "", "", status.Error(codes.Unauthenticated, "unexpected peer transport credentials")
	}

	if len(tlsAuth.State.VerifiedChains) == 0 || len(tlsAuth.State.VerifiedChains[0]) == 0 {
		log.Error().Msgf("[grpc][mTLS] Could not verify peer certificate")
		return "", "", status.Error(codes.Unauthenticated, "could not verify peer certificate")
	}

	// Check whether the subject common name is one that is allowed to connect.
	cn := tlsAuth.State.VerifiedChains[0][0].Subject.CommonName

	certificateSerialNumber := tlsAuth.State.VerifiedChains[0][0].SerialNumber.String()
	return certificate.CommonName(cn), certificate.SerialNumber(certificateSerialNumber), nil
}
