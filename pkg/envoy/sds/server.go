package sds

import (
	"context"
	"io/ioutil"
	"time"

	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	authapi "github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	sdsapi "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/gogo/protobuf/types"
	"github.com/golang/glog"
)

const (
	maxConnections = 10000

	// Save the experimental keys here
	keysDirTemp = "/tmp/keys/"

	// TODO(draychev): remove hard coded stuff
	certificateName = "server_cert"

	certFileName = "cert.pem"
	keyFileName  = "key.pem"

	typeSecret = "type.googleapis.com/envoy.api.v2.auth.Secret"
)

// Options provides all of the configuration parameters for secret discovery service.
type Options struct {
	// UDSPath is the unix domain socket through which SDS server communicates with proxies.
	UDSPath string
}

type secretItem struct {
	certificateChain []byte
	privateKey       []byte
}

// Server is the SDS server struct
type Server struct {
	// TODO: we should track more than one nonce. One nonce limits us to have only one Envoy process per SDS server.
	lastNonce string

	connectionNum int
	keysDirectory string

	// secretsManager secrets.SecretsManager

	// skipToken indicates whether token is required.
	skipToken bool

	ticker         *time.Ticker
	tickerInterval time.Duration

	// close channel.
	closing chan bool
}

// NewSDSServer creates a new SDS server
func NewSDSServer(keysDirectory *string) *Server {
	keysDir := keysDirTemp
	if keysDirectory != nil {
		keysDir = *keysDirectory
	}
	// secretsManager := secrets.SecretsManager()
	skipTokenVerification := false
	recycleInterval := 5 * time.Second

	return &Server{
		connectionNum: 0,
		keysDirectory: keysDir,

		// 	secretsManager: secretsManager,

		// TODO(draychev): implement
		skipToken: skipTokenVerification,

		tickerInterval: recycleInterval,
		closing:        make(chan bool),
	}
}

// DeltaSecrets is an SDS interface requirement
func (s *Server) DeltaSecrets(sdsapi.SecretDiscoveryService_DeltaSecretsServer) error {
	panic("NotImplemented")
}

func (s *Server) sdsDiscoveryResponse(si *secretItem, proxyID string) (*xdsapi.DiscoveryResponse, error) {
	glog.Info("[SDS] Composing SDS Discovery Response...")
	s.lastNonce = time.Now().String()
	resp := &xdsapi.DiscoveryResponse{
		TypeUrl:     typeSecret,
		VersionInfo: s.lastNonce,
		Nonce:       s.lastNonce,
	}

	secret := &authapi.Secret{
		Name: certificateName,
	}
	secret.Type = &authapi.Secret_TlsCertificate{
		TlsCertificate: &authapi.TlsCertificate{
			CertificateChain: &core.DataSource{
				Specifier: &core.DataSource_InlineBytes{
					InlineBytes: si.certificateChain,
				},
			},
			PrivateKey: &core.DataSource{
				Specifier: &core.DataSource_InlineBytes{
					InlineBytes: si.privateKey,
				},
			},
		},
	}

	ms, err := types.MarshalAny(secret)
	if err != nil {
		glog.Errorf("Failed to marshal secret for proxy %q: %v", proxyID, err)
		return nil, err
	}
    glog.V(9).Infof("SDS Response: %+v")
	resp.Resources = append(resp.Resources, ms)

	return resp, nil
}

func getSecretItem(keyDirectory string) (*secretItem, error) {
	cert, err := ioutil.ReadFile(keyDirectory + certFileName)
	if err != nil {
		glog.Infof("Failed to read cert chain from %s: %s", keyDirectory+certFileName, err)
		return nil, err
	}
	key, err := ioutil.ReadFile(keyDirectory + keyFileName)
	if err != nil {
		glog.Info("Failed to read private key", err)
		return nil, err
	}

	secret := &secretItem{
		certificateChain: cert,
		privateKey:       key,
	}

	return secret, nil
}

func (s *Server) isConnectionAllowed() error {
	if s.connectionNum >= maxConnections {
		return errTooManyConnections
	}
	s.connectionNum++
	return nil
}

// FetchSecrets fetches the certs
func (s *Server) FetchSecrets(ctx context.Context, discReq *xdsapi.DiscoveryRequest) (*xdsapi.DiscoveryResponse, error) {
	glog.Infof("Fetching Secrets...")
	secret, err := getSecretItem(s.keysDirectory)
	if err != nil {
		return nil, err
	}
	glog.Infof("Responding with Secrets...")
	return s.sdsDiscoveryResponse(secret, discReq.Node.Id)
}
