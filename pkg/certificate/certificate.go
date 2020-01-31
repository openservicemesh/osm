package certificate

import (
	"io/ioutil"

	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/logging"
)

const (
	keysDefaultDirectory = "/etc/ssl/certs/"
	certFileName         = "cert.pem"
	keyFileName          = "key.pem"
)

type Certificate struct {
	name             string
	certificateChain []byte
	privateKey       []byte
}

// GetName implements Certificater
func (c Certificate) GetName() string {
	return c.name
}

// GetCertificateChain implements Certificater
func (c Certificate) GetCertificateChain() []byte {
	return c.certificateChain
}

// GetPrivateKey implements Certificater
func (c Certificate) GetPrivateKey() []byte {
	return c.privateKey
}

func newCertificate(cn CommonName) (*Certificate, error) {
	glog.V(log.LvlTrace).Infof("[certificate] Creating a certificate for CN=%s", cn)
	// TODO(draychev): Temporarily read certificates from file until we integrate w/ KeyVault
	certificateChain, err := ioutil.ReadFile(keysDefaultDirectory + certFileName)
	if err != nil {
		glog.Infof("[certificate] Failed to read certificateChain chain from %s: %s", keysDefaultDirectory+certFileName, err)
		return nil, err
	}
	privateKey, err := ioutil.ReadFile(keysDefaultDirectory + keyFileName)
	if err != nil {
		glog.Info("[certificate] Failed to read private privateKey", err)
		return nil, err
	}

	secret := &Certificate{
		// TODO(draychev): what makes sense for the Name?
		name:             string(cn),
		certificateChain: certificateChain,
		privateKey:       privateKey,
	}

	return secret, nil
}
