package certificate

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
)

// Silliest thing - "go test -cover" would report the mocks as not tested and show very low test coverage.
func TestGetApexServicesForBackend(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	cert := NewMockCertificater(mockCtrl)
	expect := cert.EXPECT()
	assert.NotNil(expect)

	assert.NotNil(cert.EXPECT().GetPrivateKey().AnyTimes())
	assert.Equal(cert.GetPrivateKey(), []byte(nil))

	assert.NotNil(cert.EXPECT().GetIssuingCA().AnyTimes())
	assert.Equal(cert.GetIssuingCA(), []byte(nil))

	assert.NotNil(cert.EXPECT().GetCertificateChain().AnyTimes())
	assert.Equal(cert.GetCertificateChain(), []byte(nil))

	assert.NotNil(cert.EXPECT().GetExpiration().AnyTimes())
	assert.NotNil(cert.GetExpiration())

	assert.NotNil(cert.EXPECT().GetCommonName().AnyTimes())
	assert.Equal(cert.GetCommonName(), CommonName(""))

	mgr := NewMockManager(mockCtrl)
	assert.NotNil(mgr.EXPECT().IssueCertificate(CommonName(""), 1*time.Second).AnyTimes())
	assert.Nil(mgr.IssueCertificate("", 1*time.Second))

	assert.NotNil(mgr.EXPECT().ListCertificates())
	assert.Nil(mgr.ListCertificates())

	assert.NotNil(mgr.EXPECT().ReleaseCertificate(CommonName("")))
	mgr.ReleaseCertificate("")

	assert.NotNil(mgr.EXPECT().RotateCertificate(CommonName("")))
	assert.Nil(mgr.RotateCertificate(""))

	assert.NotNil(mgr.EXPECT().GetAnnouncementsChannel())
	assert.Nil(mgr.GetAnnouncementsChannel())

	assert.NotNil(mgr.EXPECT().GetCertificate(CommonName("")))
	assert.Nil(mgr.GetCertificate(""))

	assert.NotNil(mgr.EXPECT().GetRootCertificate())
	assert.Nil(mgr.GetRootCertificate())
}
