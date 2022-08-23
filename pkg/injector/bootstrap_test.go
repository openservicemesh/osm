package injector

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	tassert "github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/tests/certificates"

	"github.com/openservicemesh/osm/pkg/certificate"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy/bootstrap"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/models"
)

var _ = Describe("Test functions creating Envoy bootstrap configuration", func() {
	const (
		containerName = "-container-name-"
		envoyImage    = "-envoy-image-"
		clusterID     = "-cluster-id-"
	)

	isTrue := true
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "namespace",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: "not-me",
					Kind: "still not me",
				},
				{
					Name:       "workload-name",
					Kind:       "workload-kind",
					Controller: &isTrue,
				},
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "svcacc",
		},
	}

	namespace := "namespace"

	meshConfig := v1alpha2.MeshConfig{
		Spec: v1alpha2.MeshConfigSpec{
			Sidecar: v1alpha2.SidecarSpec{
				TLSMinProtocolVersion: "TLSv1_2",
				TLSMaxProtocolVersion: "TLSv1_3",
				CipherSuites:          []string{},
				EnvoyWindowsImage:     envoyImage,
				EnvoyImage:            envoyImage,
				LogLevel:              "debug",
				Resources: corev1.ResourceRequirements{
					// Test set Limits
					Limits: map[corev1.ResourceName]resource.Quantity{
						"cpu":    resource.MustParse("2"),
						"memory": resource.MustParse("512M"),
					},
				},
			},
		},
	}

	originalHealthProbes := map[string]models.HealthProbes{
		"my-container": {
			Liveness:  &models.HealthProbe{Path: "/liveness", Port: 81},
			Readiness: &models.HealthProbe{Path: "/readiness", Port: 82},
			Startup:   &models.HealthProbe{Path: "/startup", Port: 83},
		},
		"my-sidecar": {
			Liveness:  &models.HealthProbe{Path: "/liveness", Port: 84},
			Readiness: &models.HealthProbe{Path: "/readiness", Port: 85},
			Startup:   &models.HealthProbe{Path: "/startup", Port: 86},
		},
	}

	expectedRewrittenContainerPorts := []corev1.ContainerPort{
		{Name: "proxy-admin", HostPort: 0, ContainerPort: 15000, Protocol: "", HostIP: ""},
		{Name: "proxy-inbound", HostPort: 0, ContainerPort: 15003, Protocol: "", HostIP: ""},
		{Name: "proxy-metrics", HostPort: 0, ContainerPort: 15010, Protocol: "", HostIP: ""},
		{Name: "liveness-port", HostPort: 0, ContainerPort: 15901, Protocol: "", HostIP: ""},
		{Name: "readiness-port", HostPort: 0, ContainerPort: 15902, Protocol: "", HostIP: ""},
		{Name: "startup-port", HostPort: 0, ContainerPort: 15903, Protocol: "", HostIP: ""},
	}

	Context("Test getEnvoyContainerPorts()", func() {
		It("creates container port list", func() {
			actualRewrittenContainerPorts := getEnvoyContainerPorts(originalHealthProbes)
			Expect(actualRewrittenContainerPorts).To(Equal(expectedRewrittenContainerPorts))
		})
	})

	Context("test unix getEnvoySidecarContainerSpec()", func() {
		It("creates Envoy sidecar spec", func() {
			actual := getEnvoySidecarContainerSpec(pod, namespace, meshConfig, originalHealthProbes, constants.OSLinux)

			expected := corev1.Container{
				Name:            constants.EnvoyContainerName,
				Image:           envoyImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				SecurityContext: &corev1.SecurityContext{
					AllowPrivilegeEscalation: pointer.BoolPtr(false),
					RunAsUser: func() *int64 {
						uid := constants.EnvoyUID
						return &uid
					}(),
				},
				Ports: expectedRewrittenContainerPorts,
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      envoyBootstrapConfigVolume,
						ReadOnly:  true,
						MountPath: bootstrap.EnvoyProxyConfigPath,
					},
				},
				Resources: corev1.ResourceRequirements{
					// Test set Limits
					Limits: map[corev1.ResourceName]resource.Quantity{
						"cpu":    resource.MustParse("2"),
						"memory": resource.MustParse("512M"),
					},
					// Test unset Requests
					Requests: nil,
				},
				Command: []string{
					"envoy",
				},
				Args: []string{
					"--log-level", "debug",
					"--config-path", "/etc/envoy/bootstrap.yaml",
					"--service-cluster", "svcacc.namespace",
				},
				Env: []corev1.EnvVar{
					{
						Name:  "POD_UID",
						Value: "",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "",
								FieldPath:  "metadata.uid",
							},
							ResourceFieldRef: nil,
							ConfigMapKeyRef:  nil,
							SecretKeyRef:     nil,
						},
					},
					{
						Name:  "POD_NAME",
						Value: "",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "",
								FieldPath:  "metadata.name",
							},
							ResourceFieldRef: nil,
							ConfigMapKeyRef:  nil,
							SecretKeyRef:     nil,
						},
					},
					{
						Name:  "POD_NAMESPACE",
						Value: "",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "",
								FieldPath:  "metadata.namespace",
							},
							ResourceFieldRef: nil,
							ConfigMapKeyRef:  nil,
							SecretKeyRef:     nil,
						},
					},
					{
						Name:  "POD_IP",
						Value: "",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "",
								FieldPath:  "status.podIP",
							},
							ResourceFieldRef: nil,
							ConfigMapKeyRef:  nil,
							SecretKeyRef:     nil,
						},
					},
					{
						Name:  "SERVICE_ACCOUNT",
						Value: "",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "",
								FieldPath:  "spec.serviceAccountName",
							},
							ResourceFieldRef: nil,
							ConfigMapKeyRef:  nil,
							SecretKeyRef:     nil,
						},
					},
				},
			}

			Expect(actual).To(Equal(expected))
		})
	})

	Context("test Windows getEnvoySidecarContainerSpec()", func() {
		It("creates Envoy sidecar spec", func() {
			actual := getEnvoySidecarContainerSpec(pod, namespace, meshConfig, originalHealthProbes, constants.OSWindows)

			expected := corev1.Container{
				Name:            constants.EnvoyContainerName,
				Image:           envoyImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				SecurityContext: &corev1.SecurityContext{
					WindowsOptions: &corev1.WindowsSecurityContextOptions{
						RunAsUserName: func() *string {
							userName := "EnvoyUser"
							return &userName
						}(),
					},
				},
				Ports: expectedRewrittenContainerPorts,
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      envoyBootstrapConfigVolume,
						ReadOnly:  true,
						MountPath: bootstrap.EnvoyProxyConfigPath,
					},
				},
				Resources: corev1.ResourceRequirements{
					// Test set Limits
					Limits: map[corev1.ResourceName]resource.Quantity{
						"cpu":    resource.MustParse("2"),
						"memory": resource.MustParse("512M"),
					},
					// Test unset Requests
					Requests: nil,
				},
				Command: []string{
					"envoy",
				},
				Args: []string{
					"--log-level", "debug",
					"--config-path", "/etc/envoy/bootstrap.yaml",
					"--service-cluster", "svcacc.namespace",
				},
				Env: []corev1.EnvVar{
					{
						Name:  "POD_UID",
						Value: "",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "",
								FieldPath:  "metadata.uid",
							},
							ResourceFieldRef: nil,
							ConfigMapKeyRef:  nil,
							SecretKeyRef:     nil,
						},
					},
					{
						Name:  "POD_NAME",
						Value: "",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "",
								FieldPath:  "metadata.name",
							},
							ResourceFieldRef: nil,
							ConfigMapKeyRef:  nil,
							SecretKeyRef:     nil,
						},
					},
					{
						Name:  "POD_NAMESPACE",
						Value: "",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "",
								FieldPath:  "metadata.namespace",
							},
							ResourceFieldRef: nil,
							ConfigMapKeyRef:  nil,
							SecretKeyRef:     nil,
						},
					},
					{
						Name:  "POD_IP",
						Value: "",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "",
								FieldPath:  "status.podIP",
							},
							ResourceFieldRef: nil,
							ConfigMapKeyRef:  nil,
							SecretKeyRef:     nil,
						},
					},
					{
						Name:  "SERVICE_ACCOUNT",
						Value: "",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "",
								FieldPath:  "spec.serviceAccountName",
							},
							ResourceFieldRef: nil,
							ConfigMapKeyRef:  nil,
							SecretKeyRef:     nil,
						},
					},
				},
			}

			Expect(actual).To(Equal(expected))
		})
	})
})

func TestListBootstrapSecrets(t *testing.T) {
	testCases := []struct {
		name       string
		secrets    []*corev1.Secret
		expSecrets []*corev1.Secret
	}{
		{
			name: "get bootstrap secrets from k8s secrets",
			secrets: []*corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "notBootstrapSecret",
						Namespace: "testNamespace",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      bootstrapSecretPrefix + "proxyUUID",
						Namespace: "testNamespace",
					},
				},
			},
			expSecrets: []*corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      bootstrapSecretPrefix + "proxyUUID",
						Namespace: "testNamespace",
					},
				},
			},
		},
		{
			name: "no bootstrap secrets in k8s secrets",
			secrets: []*corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "notBootstrapSecret",
						Namespace: "testNamespace",
					},
				},
			},
			expSecrets: []*corev1.Secret{},
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Running test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			mockController := k8s.NewMockController(gomock.NewController(t))
			mockController.EXPECT().ListSecrets().Return(tc.secrets)

			certManager, err := certificate.FakeCertManager()
			assert.Nil(err)

			b := NewBootstrapSecretRotator(mockController, certManager, time.Duration(1))

			actual := b.listBootstrapSecrets()
			assert.ElementsMatch(tc.expSecrets, actual)
		})
	}
}

func TestRotateBootstrapSecrets(t *testing.T) {
	assert := tassert.New(t)

	testNs := "testNamespace"
	proxyUUID := uuid.New()
	secretName := bootstrapSecretPrefix + proxyUUID.String()

	testCases := []struct {
		name     string
		certName string
		secrets  []*corev1.Secret
	}{
		{
			name:     "update bootstrap secret with new cert",
			certName: certificate.CommonName(proxyUUID.String() + ".test.cert").String(),
			secrets: []*corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: testNs,
					},
					Data: map[string][]byte{
						bootstrap.EnvoyXDSCACertFile: {},
						bootstrap.EnvoyXDSCertFile:   []byte(certificates.SampleCertificatePEM),
						bootstrap.EnvoyXDSKeyFile:    []byte(certificates.SamplePrivateKeyPEM),
					},
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Running test case %d: %s", i, tc.name), func(t *testing.T) {
			certManager, err := certificate.FakeCertManager()
			assert.Nil(err)

			mockController := k8s.NewMockController(gomock.NewController(t))
			mockController.EXPECT().ListSecrets().Return(tc.secrets)

			cert, err := certManager.IssueCertificate(tc.certName, certificate.Internal)
			assert.Nil(err)

			secretData := map[string][]byte{
				bootstrap.EnvoyXDSCACertFile: cert.GetTrustedCAs(),
				bootstrap.EnvoyXDSCertFile:   cert.GetCertificateChain(),
				bootstrap.EnvoyXDSKeyFile:    cert.GetPrivateKey(),
			}
			mockController.EXPECT().UpdateSecret(context.Background(), tc.secrets[0], secretData)

			bootstrapSecretRotator := NewBootstrapSecretRotator(mockController, certManager, time.Duration(1))
			bootstrapSecretRotator.rotateBootstrapSecrets(context.Background())
		})
	}
}

func TestGetCert(t *testing.T) {
	testCases := []struct {
		name    string
		secret  *corev1.Secret
		expCert bool
		expErr  error
	}{
		{
			name: "valid bootstrap secret",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "bootstrapSecret",
				},
				Data: map[string][]byte{
					bootstrap.EnvoyXDSCACertFile: {},
					bootstrap.EnvoyXDSCertFile:   []byte(certificates.SampleCertificatePEM),
					bootstrap.EnvoyXDSKeyFile:    {},
				},
			},
			expCert: true,
			expErr:  nil,
		},
		{
			name: "invalid bootstrap secret - missing cert field - sds_cert.pem",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "invalidSecret",
				},
				Data: map[string][]byte{
					bootstrap.EnvoyXDSCACertFile: {},
					bootstrap.EnvoyXDSKeyFile:    {},
				},
			},
			expCert: false,
			expErr:  certificate.ErrInvalidCertSecret,
		},
		{
			name: "invalid bootstrap secret - missing cert field - sds_key.pem",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "invalidSecret",
				},
				Data: map[string][]byte{
					bootstrap.EnvoyXDSCertFile:   {},
					bootstrap.EnvoyXDSCACertFile: {},
				},
			},
			expCert: false,
			expErr:  certificate.ErrInvalidCertSecret,
		},
		{
			name: "unable to decode PEM",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "invalidSecret",
				},
				Data: map[string][]byte{
					bootstrap.EnvoyXDSCertFile:   {},
					bootstrap.EnvoyXDSCACertFile: {},
					bootstrap.EnvoyXDSKeyFile:    {},
				},
			},
			expCert: false,
			expErr:  certificate.ErrNoCertificateInPEM,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Running test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)
			cert, err := getCert(tc.secret)
			if tc.expCert {
				assert.NotNil(cert)
			} else {
				assert.Nil(cert)
			}
			assert.Equal(tc.expErr, err)
		})
	}
}
