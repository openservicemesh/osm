package certificate

import (
	"fmt"
	time "time"
)

var (
	caCert = &Certificate{
		CommonName: "Test CA",
		Expiration: time.Now().Add(time.Hour * 24),
	}
	validity = time.Hour
)

type fakeIssuer struct{}

// IssueCertificate is a testing helper to satisfy the certificate client interface
func (i *fakeIssuer) IssueCertificate(cn CommonName, validityPeriod time.Duration) (*Certificate, error) {
	return &Certificate{
		CommonName: cn,
		Expiration: time.Now().Add(validityPeriod),
	}, nil
}

// FakeCertManager is a testing helper that returns a *certificate.Manager
func FakeCertManager() (*Manager, error) {
	cm, err := NewManager(
		caCert,
		&fakeIssuer{},
		validity,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating fakeCertManager, err: %w", err)
	}
	return cm, nil
}
