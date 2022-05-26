package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Test health probes can succeed",
	OSMDescribeInfo{
		Tier:   1,
		Bucket: 4,
	},
	func() {
		tls := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "nginx-tls",
			},
			Type: corev1.SecretTypeTLS,
			StringData: map[string]string{
				"tls.crt": `-----BEGIN CERTIFICATE-----
MIIEpDCCAowCCQD19P4QhwJXHDANBgkqhkiG9w0BAQsFADAUMRIwEAYDVQQDDAls
b2NhbGhvc3QwHhcNMjEwNDA3MTU0ODM5WhcNMjIwNDA3MTU0ODM5WjAUMRIwEAYD
VQQDDAlsb2NhbGhvc3QwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQCq
OMKeB4+F6LmVT1PPrTO+VLvioivqOM3dc7CVYBTrOoB2QDh9LXWPBb9ct/n4mqeD
npJnAa91KENhgMrhCDnkR7YvswULyfdsnUe96QPsONvGewSkmCzdsNwhVaVWOpUE
6R9mC04b8XK32xVII78MTPMPkVd4YGu99XwDD9fiJ3kp7Ftqsskswsa0t4pGRXkr
UjEZ2Lp+hAKWYoioAj9SAGS1xAXlEPuwXy/3YUo+BaV5uLvFHuinmD6E1CrYhXJe
9/RVAoRb1kbZmnrBWLdrt0K0OWPimPcENDwOXVeO4tWMF0MJZGLDM73nCpLbu893
t95/E7qzDlbvsQ05NpejcqCL95XCI8kmk9ZqkOOpBDJiixYFE7BvLtol49N5vqAr
nynAIVY4iZtk1TpmnEF0Cxx6/VXj2pvzo7Pgm42S29NrMXoIZpgiPtJAK/PH2kE2
LCAneDmoJOc6hn56T4DSfnjBtoWrXhEdoeYNEFRp2+Zf1evWMu4XBbEFMswHOXmI
d0u4XEBV8ROoZolfh2TTLVyWSTd02st7Dtw+OK4C6LPdhSYjhHsz+KG8yHEC/zxV
O+9NDohsw83/xI90wKQ5gS9GauKanxiEzYSTXcFGs3lltsavXqLM8ukiAqdFn5Jp
7ObuusGX3Pi5T7ABbpt3zqCk7FPISgqg8o1m4LCjfQIDAQABMA0GCSqGSIb3DQEB
CwUAA4ICAQCcTkGMGlZClAd7dxt8g4ZHUr+2RwY+LncDoN9K8060cxsmPR+KUBU/
mOzZoJXeUyvL56UsjmXkUfUxIrX3Erkb8KTKk5f1J71TeYTkaOcXiXWAPWwuOoTu
yn88EDb2LHwBxVsIkLQXUhrurKgnuyvlGrUnZxTz7wXVxYo5Tn4G8izca1VC2FBb
oF7yhhTujnfSvtLuMMgGhFoL7YPB/kNMZpn+Jgs4Qc7fWYrgfw4Xu7kzb85eOMJm
ExEU8hNEsPX16DMpyaUq8qROMfyPsFq88NyAEj/8SyuEun1m9dO3PymEW28cWtzO
ZIwSRwZrod+od0E5Ic0a9Nboq5KX1N0ShfrTJL3BcpDnl8r3wxQdXg0h+C2YlH11
EfQloMfKIsOFzM6dg5VQQNxzJdAKC/vHAEClUCdwcjl15TsoP5X+BWxZ8kBxoCLu
yvDqJWQIK6j2lD1TF8IgqFfF9sbyvVBEcffp3xxswXA1gWKmCFoYLZOe1oIrUP46
k0KOU5Gl9P3HtiNDYV5tdNQZj2yM7x3273bdGn991+9U+9rpeqSJg4HD1p20ATOI
JyFx3hr65q1hMf/PZ3u+tyUdfyxgu+wXf9UgOaFxLqQjdRPqUOolnh5kKupiCy38
F0GFx++vOgfecOyhkpKkFb1S23XB69wlWTX/qrktHMILRyW1Eb2DgA==
-----END CERTIFICATE-----
`,
				"tls.key": `-----BEGIN PRIVATE KEY-----
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
ZRaHimUjXwDqCTKcz1ttlH6SRcfbhI5D
-----END PRIVATE KEY-----
`,
			},
		}

		conf := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "nginx-conf",
			},
			Data: map[string]string{
				"default.conf": `
server {
listen       80;
listen       443 ssl;
ssl_certificate /etc/nginx//tls/tls.crt;
ssl_certificate_key /etc/nginx//tls/tls.key;
server_name  localhost;

location / {
	root   /usr/share/nginx/html;
	index  index.html index.htm;
}

# redirect server error pages to the static page /50x.html
#
error_page   500 502 503 504  /50x.html;
location = /50x.html {
	root   /usr/share/nginx/html;
}
}
`,
			},
		}

		http := &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/",
					Port:   intstr.FromInt(80),
					Scheme: corev1.URISchemeHTTP,
				},
			},
		}

		https := &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/",
					Port:   intstr.FromInt(443),
					Scheme: corev1.URISchemeHTTPS,
				},
			},
		}

		tcp := &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt(80),
				},
			},
		}

		incorrectTCP := &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt(81),
				},
			},
		}

		makePod := func(name string, probe *corev1.Probe) *corev1.Pod {
			podDef := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.19-alpine",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "tls",
									MountPath: "/etc/nginx/tls",
									ReadOnly:  true,
								},
								{
									Name:      "config",
									MountPath: "/etc/nginx/conf.d",
									ReadOnly:  true,
								},
							},
							StartupProbe:   probe,
							LivenessProbe:  probe,
							ReadinessProbe: probe,
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "tls",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "nginx-tls",
								},
							},
						},
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "nginx-conf",
									},
								},
							},
						},
					},
				},
			}

			if Td.AreRegistryCredsPresent() {
				podDef.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
					{
						Name: RegistrySecretName,
					},
				}
			}

			return podDef
		}

		It("Configures Pods' probes so they work as expected", func() {
			const ns = "healthprobe"
			// Install OSM
			Expect(Td.InstallOSM(Td.GetOSMInstallOpts())).To(Succeed())

			Expect(Td.CreateNs(ns, nil)).To(Succeed())
			Expect(Td.AddNsToMesh(true, ns)).To(Succeed())

			pods := []*corev1.Pod{
				makePod("nginx-http", http),
				makePod("nginx-https", https),
				makePod("nginx-tcp", tcp),
			}

			_, err := Td.Client.CoreV1().Secrets(ns).Create(context.TODO(), tls, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			_, err = Td.Client.CoreV1().ConfigMaps(ns).Create(context.TODO(), conf, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			for _, pod := range pods {
				_, err = Td.Client.CoreV1().Pods(ns).Create(context.TODO(), pod, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
			}

			// Expect it to be up and running in it's receiver namespace
			Expect(Td.WaitForPodsRunningReady(ns, 60*time.Second, len(pods), nil)).To(Succeed())
		})

		It("Incorrectly configures Pods' TCPSocket probes so they fail as expected", func() {
			const ns = "healthprobe"
			// Install OSM
			Expect(Td.InstallOSM(Td.GetOSMInstallOpts())).To(Succeed())

			Expect(Td.CreateNs(ns, nil)).To(Succeed())
			Expect(Td.AddNsToMesh(true, ns)).To(Succeed())

			pod := makePod("nginx-tcp", incorrectTCP)

			_, err := Td.Client.CoreV1().Secrets(ns).Create(context.TODO(), tls, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			_, err = Td.Client.CoreV1().ConfigMaps(ns).Create(context.TODO(), conf, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			_, err = Td.Client.CoreV1().Pods(ns).Create(context.TODO(), pod, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Expect it to not be ready
			timeout := 60 * time.Second
			Expect(Td.WaitForPodsRunningReady(ns, timeout, 1, nil)).To(MatchError(fmt.Sprintf("not all pods were Running & Ready in NS healthprobe after %v", timeout)))
		})

	},
)
