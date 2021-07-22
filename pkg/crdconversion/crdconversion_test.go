package crdconversion

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/tests"
)

type mockCertificate struct{}

func (mc mockCertificate) GetCommonName() certificate.CommonName     { return "" }
func (mc mockCertificate) GetCertificateChain() []byte               { return []byte("chain") }
func (mc mockCertificate) GetPrivateKey() []byte                     { return []byte("key") }
func (mc mockCertificate) GetIssuingCA() []byte                      { return []byte("ca") }
func (mc mockCertificate) GetExpiration() time.Time                  { return time.Now() }
func (mc mockCertificate) GetSerialNumber() certificate.SerialNumber { return "serial_number" }

func TestUpdateCrdConversionWebhookConfiguration(t *testing.T) {
	assert := tassert.New(t)
	cert := mockCertificate{}

	crdClient := fake.NewSimpleClientset(&apiv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "tests.test.openservicemesh.io",
		},
		Spec: apiv1.CustomResourceDefinitionSpec{
			Group: "test.openservicemesh.io",
			Names: apiv1.CustomResourceDefinitionNames{
				Plural:   "tests",
				Singular: "test",
				Kind:     "test",
				ListKind: "testList",
			},
			Scope: apiv1.NamespaceScoped,
			Versions: []apiv1.CustomResourceDefinitionVersion{{
				Name:    "v1alpha1",
				Served:  true,
				Storage: true,
				Schema: &apiv1.CustomResourceValidation{
					OpenAPIV3Schema: &apiv1.JSONSchemaProps{
						Type:       "object",
						Properties: make(map[string]apiv1.JSONSchemaProps),
					},
				},
			}},
		},
	})

	err := updateCrdConversionWebhookConfiguration(cert, crdClient.ApiextensionsV1(), tests.Namespace, "tests.test.openservicemesh.io", "/testconversion")
	assert.Nil(err)

	crds, err := crdClient.ApiextensionsV1().CustomResourceDefinitions().List(context.TODO(), metav1.ListOptions{})
	assert.Nil(err)

	crd := crds.Items[0]
	assert.Equal(crd.Spec.Conversion.Strategy, apiv1.WebhookConverter)
	assert.Equal(crd.Spec.Conversion.Webhook.ClientConfig.CABundle, []byte("chain"))
	assert.Equal(crd.Spec.Conversion.Webhook.ClientConfig.Service.Namespace, tests.Namespace)
	assert.Equal(crd.Spec.Conversion.Webhook.ClientConfig.Service.Name, crdConverterServiceName)
	assert.Equal(crd.Spec.Conversion.Webhook.ConversionReviewVersions, conversionReviewVersions)
}

func TestNewConversionWebhook(t *testing.T) {
	assert := tassert.New(t)
	crdConversionConfig := Config{}
	crdClient := fake.NewSimpleClientset()
	kubeClient := k8sfake.NewSimpleClientset()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	keySize := 2048
	mockConfigurator.EXPECT().GetCertKeyBitSize().Return(keySize).AnyTimes()
	fakeCertManager := tresor.NewFakeCertManager(mockConfigurator)
	osmNamespace := "-osm-namespace-"
	stop := make(<-chan struct{})

	actualErr := NewConversionWebhook(crdConversionConfig, kubeClient, crdClient.ApiextensionsV1(), fakeCertManager, osmNamespace, stop)
	assert.NotNil(actualErr)
}
