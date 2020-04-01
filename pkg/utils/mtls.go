package utils

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/open-service-mesh/osm/pkg/certificate"
)

func setupMutualTLS(insecure bool, serverName string, certPem string, keyPem string, rootCertPem string) grpc.ServerOption {
	certif, err := tls.LoadX509KeyPair(certPem, keyPem)
	if err != nil {
		log.Fatal().Err(err).Msgf("[grpc][mTLS][%s] Failed loading Certificate (%+v) and Key (%+v) PEM files", serverName, certPem, keyPem)
	}

	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(rootCertPem)
	if err != nil {
		log.Fatal().Err(err).Msgf("[grpc][mTLS][%s] Failed to read client CA cert from %s", serverName, rootCertPem)
	}

	// Load the set of Root CAs
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		log.Fatal().Msgf("[grpc][mTLS][%s] Filed to append client certs.", serverName)
	}

	tlsConfig := tls.Config{
		InsecureSkipVerify: insecure,
		ServerName:         serverName,
		ClientAuth:         tls.RequireAndVerifyClientCert,
		Certificates:       []tls.Certificate{certif},
		ClientCAs:          certPool,
	}
	return grpc.Creds(credentials.NewTLS(&tlsConfig))
}

// ValidateClient ensures that the connected client is authorized to connect to the gRPC server.
func ValidateClient(ctx context.Context, allowedCommonNames map[string]interface{}, serverName string) (certificate.CommonName, error) {
	mtlsPeer, ok := peer.FromContext(ctx)
	if !ok {
		log.Error().Msgf("[grpc][mTLS][%s] No peer found", serverName)
		return "", status.Error(codes.Unauthenticated, "no peer found")
	}

	tlsAuth, ok := mtlsPeer.AuthInfo.(credentials.TLSInfo)
	if !ok {
		log.Error().Msgf("[grpc][mTLS][%s] Unexpected peer transport credentials.", serverName)
		return "", status.Error(codes.Unauthenticated, "unexpected peer transport credentials")
	}

	if len(tlsAuth.State.VerifiedChains) == 0 || len(tlsAuth.State.VerifiedChains[0]) == 0 {
		log.Error().Msgf("[grpc][mTLS][%s] Could not verify peer certificate.", serverName)
		return "", status.Error(codes.Unauthenticated, "could not verify peer certificate")
	}

	// Check whether the subject common name is one that is allowed to connect.
	cn := tlsAuth.State.VerifiedChains[0][0].Subject.CommonName
	if _, ok := allowedCommonNames[cn]; len(allowedCommonNames) > 0 && !ok {
		log.Error().Msgf("[grpc][mTLS][%s] Subject common name %+v not allowed", serverName, cn)
		return "", status.Error(codes.Unauthenticated, "disallowed subject common name")
	}
	return certificate.CommonName(cn), nil
}
