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
	"github.com/openservicemesh/osm/pkg/certificate/pem"
	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestUpdateCrdConfiguration(t *testing.T) {
	testCases := []struct {
		name string
	}{
		{
			name: "base case",
		},
	}

	var cert = &certificate.Certificate{
		CommonName:   "",
		CertChain:    pem.Certificate("chain"),
		PrivateKey:   pem.PrivateKey("key"),
		IssuingCA:    pem.RootCertificate("ca"),
		TrustedCAs:   pem.RootCertificate("ca"),
		Expiration:   time.Now(),
		SerialNumber: "serial_number",
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			crdClient := fake.NewSimpleClientset(&apiv1.CustomResourceDefinition{
				TypeMeta: metav1.TypeMeta{
					Kind:       "CustomResourceDefinition",
					APIVersion: "apiextensions.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "tests.test.openservicemesh.io",
					Labels: map[string]string{
						constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue,
					},
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

			err := updateCrdConfiguration(cert, crdClient.ApiextensionsV1(), tests.Namespace, "tests.test.openservicemesh.io", "/testconversion")
			assert.Nil(err)

			crds, err := crdClient.ApiextensionsV1().CustomResourceDefinitions().List(context.TODO(), metav1.ListOptions{})
			assert.Nil(err)

			crd := crds.Items[0]
			assert.Equal(crd.Spec.Conversion.Strategy, apiv1.WebhookConverter)
			assert.Equal(crd.Spec.Conversion.Webhook.ClientConfig.CABundle, []byte("ca"))
			assert.Equal(crd.Spec.Conversion.Webhook.ClientConfig.Service.Namespace, tests.Namespace)
			assert.Equal(crd.Spec.Conversion.Webhook.ClientConfig.Service.Name, constants.OSMBootstrapName)
			assert.Equal(crd.Spec.Conversion.Webhook.ConversionReviewVersions, conversionReviewVersions)

			assert.Equal(crd.Labels[constants.OSMAppNameLabelKey], constants.OSMAppNameLabelValue)
		})
	}
}

func TestNewConversionWebhook(t *testing.T) {
	assert := tassert.New(t)
	crdClient := fake.NewSimpleClientset()
	kubeClient := k8sfake.NewSimpleClientset()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	fakeCertManager := tresorFake.NewFake(nil, 1*time.Hour)
	osmNamespace := "-osm-namespace-"
	enablesReconciler := false

	actualErr := NewConversionWebhook(context.Background(), kubeClient, crdClient.ApiextensionsV1(), fakeCertManager, osmNamespace, enablesReconciler)
	assert.NotNil(actualErr)
}
