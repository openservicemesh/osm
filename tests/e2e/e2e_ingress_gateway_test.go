package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/constants"
	. "github.com/openservicemesh/osm/tests/framework"
)

// Since the ingress gateway is a black box, we only care that the certificate we provision for it is valid.
// We can test this with a normal httpbin pod, that consumes the certificate.
var _ = OSMDescribe("Test proxy resource setting",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 8,
	},
	func() {
		var (
			serverApp  = "server"
			serverNS   = "server-ns"
			ingressApp = "ingress-gateway"
			ingressNS  = "ingress-ns"
			secretName = "ingress-gateway-cert"
		)
		Context("ingress gateway cert", func() {
			It("Tests provisioned ingress gateway cert can connect successfully", func() {
				installOpts := Td.GetOSMInstallOpts()
				installOpts.EnablePermissiveMode = false

				Expect(Td.InstallOSM(installOpts)).To(Succeed())

				Expect(Td.CreateNs(serverNS, nil)).To(Succeed())
				Expect(Td.CreateNs(ingressNS, nil)).To(Succeed())
				// Only inject the server ns.
				Expect(Td.AddNsToMesh(true, serverNS)).To(Succeed())

				meshConfig, err := Td.GetMeshConfig(Td.OsmNamespace)
				Expect(err).ToNot(HaveOccurred())

				// Create a gateway cert.
				meshConfig.Spec.Certificate.IngressGateway = &configv1alpha2.IngressGatewayCertSpec{
					SubjectAltNames:  []string{"myhost.test.com"},
					ValidityDuration: "24h",
					Secret: corev1.SecretReference{
						Name:      secretName,
						Namespace: ingressNS,
					},
				}

				_, err = Td.UpdateOSMConfig(meshConfig)
				Expect(err).NotTo(HaveOccurred())

				// Create the server App
				serverAcc, serverDeployment, serverSvc, err := Td.SimpleDeploymentApp(
					SimpleDeploymentAppDef{
						DeploymentName:     serverApp,
						Namespace:          serverNS,
						ServiceAccountName: serverApp,
						ServiceName:        serverApp,
						ReplicaCount:       1,
						Image:              "simonkowallik/httpbin",
						Ports:              []int{DefaultUpstreamServicePort},
						AppProtocol:        constants.ProtocolHTTP,
						Command:            HttpbinCmd,
						OS:                 Td.ClusterOS,
					})
				Expect(err).NotTo(HaveOccurred())

				_, err = Td.CreateServiceAccount(serverNS, &serverAcc)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateDeployment(serverNS, serverDeployment)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateService(serverNS, serverSvc)
				Expect(err).NotTo(HaveOccurred())

				// Create the ingress gateway.
				ingressAcc, ingressDeployment, ingressSvc, err := Td.SimpleDeploymentApp(SimpleDeploymentAppDef{
					DeploymentName:     ingressApp,
					Namespace:          ingressNS,
					ServiceAccountName: ingressApp,
					ServiceName:        ingressApp,
					ContainerName:      ingressApp,
					ReplicaCount:       1,
					Image:              "curlimages/curl",
					Ports:              []int{DefaultUpstreamServicePort},
					AppProtocol:        constants.ProtocolHTTP,
					Command:            []string{"sleep", "365d"},
					OS:                 Td.ClusterOS,
				})
				Expect(err).NotTo(HaveOccurred())

				ingressDeployment.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
					{
						Name:      "cert",
						MountPath: "/etc/ingress-certs",
					},
				}

				ingressDeployment.Spec.Template.Spec.Volumes = []corev1.Volume{
					{
						Name: "cert",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: secretName,
							},
						},
					},
				}

				_, err = Td.CreateServiceAccount(ingressNS, &ingressAcc)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateDeployment(ingressNS, ingressDeployment)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateService(ingressNS, ingressSvc)
				Expect(err).NotTo(HaveOccurred())

				// create an ingress backend
				ingressBackend := &policyv1alpha1.IngressBackend{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "httpbin-http",
						Namespace: serverNS,
					},
					Spec: policyv1alpha1.IngressBackendSpec{
						Backends: []policyv1alpha1.BackendSpec{
							{
								Name: serverApp,
								Port: policyv1alpha1.PortSpec{
									Number:   DefaultUpstreamServicePort,
									Protocol: "https",
								},
							},
						},
						Sources: []policyv1alpha1.IngressSourceSpec{
							{
								Kind:      "Service",
								Name:      ingressApp,
								Namespace: ingressNS,
							},
							{
								Kind: "AuthenticatedPrincipal",
								Name: fmt.Sprintf("%s.%s.cluster.local", ingressApp, ingressNS),
							},
						},
					},
				}

				Expect(Td.WaitForPodsRunningReady(serverNS, 200*time.Second, 1, nil)).To(Succeed())
				Expect(Td.WaitForPodsRunningReady(ingressNS, 200*time.Second, 1, nil)).To(Succeed())

				_, err = Td.PolicyClient.PolicyV1alpha1().IngressBackends(ingressBackend.Namespace).Create(context.TODO(), ingressBackend, metav1.CreateOptions{})
				Expect(err).ToNot((HaveOccurred()))

				ingressPods, err := Td.Client.CoreV1().Pods(ingressNS).List(context.Background(), metav1.ListOptions{})
				Expect(err).To(BeNil())

				svc, err := Td.Client.CoreV1().Services(serverNS).Get(context.Background(), serverApp, metav1.GetOptions{})
				Expect(err).To(BeNil())

				By("connecting to ingress backend via mTLS")
				success := Td.WaitForRepeatedSuccess(func() bool {
					// Get results
					result := Td.HTTPRequest(HTTPRequestDef{
						SourceNs:        ingressNS,
						SourcePod:       ingressPods.Items[0].Name,
						SourceContainer: ingressApp,

						// Note this service doesn't resolve to anything so we add a resolve line below.
						Destination: fmt.Sprintf("https://%s.%s.cluster.local:%d", serverApp, serverNS, DefaultUpstreamServicePort),
						ExtraArgs: []string{
							"--cert /etc/ingress-certs/tls.crt --key /etc/ingress-certs/tls.key --cacert /etc/ingress-certs/ca.crt",
							// Because the SAN doesn't match the svc, we need to change the service but tell curl how to resolve it.
							fmt.Sprintf("--resolve https://%s.%s.cluster.local:%d:%s", serverApp, serverNS, DefaultUpstreamServicePort, svc.Spec.ClusterIP),
						},
					})

					if result.Err != nil || result.StatusCode != 200 {
						Td.T.Logf("Failed to connect to ingress backend on pod %s: %v; received status %d", ingressPods.Items[0].Name, result.Err, result.StatusCode)
						return false
					}

					return true
				}, 5, 150*time.Second)

				Expect(success).To(BeTrue())
			})
		})
	})
