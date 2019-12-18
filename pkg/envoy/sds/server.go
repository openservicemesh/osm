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
	maxConnections = 1

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

type SDSServer struct {
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

func NewSDSServer(keysDirectory *string) *SDSServer {
	keysDir := keysDirTemp
	if keysDirectory != nil {
		keysDir = *keysDirectory
	}
	// secretsManager := secrets.SecretsManager()
	skipTokenVerification := false
	recycleInterval := 5 * time.Second

	return &SDSServer{
		connectionNum: 0,
		keysDirectory: keysDir,

		// 	secretsManager: secretsManager,

		// TODO(draychev): implement
		skipToken: skipTokenVerification,

		tickerInterval: recycleInterval,
		closing:        make(chan bool),
	}
}

func (s *SDSServer) DeltaSecrets(sdsapi.SecretDiscoveryService_DeltaSecretsServer) error {
	panic("NotImplemented")
}

func (s *SDSServer) sdsDiscoveryResponse(si *secretItem, proxyID string) (*xdsapi.DiscoveryResponse, error) {
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

func (s *SDSServer) isConnectionAllowed() error {
	if s.connectionNum >= maxConnections {
		return errTooManyConnections
	}
	s.connectionNum++
	return nil
}

func (s *SDSServer) FetchSecrets(ctx context.Context, discReq *xdsapi.DiscoveryRequest) (*xdsapi.DiscoveryResponse, error) {
	glog.Infof("Fetching Secrets...")
	secret, err := getSecretItem(s.keysDirectory)
	if err != nil {
		return nil, err
	}
	glog.Infof("Responding with Secrets...")
	return s.sdsDiscoveryResponse(secret, discReq.Node.Id)
}
