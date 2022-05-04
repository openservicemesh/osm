package injector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	corev1 "k8s.io/api/core/v1"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"

	"github.com/openservicemesh/osm/pkg/configurator"
)

var _ = Describe("Test functions creating Envoy bootstrap configuration", func() {
	const (
		containerName  = "-container-name-"
		containerImage = "-init-container-image-"
	)

	privilegedFalse := false
	runAsNonRootFalse := false
	runAsUserID := int64(0)

	mockCtrl := gomock.NewController(GinkgoT())
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	Context("test getInitContainerSpec()", func() {
		It("Creates init container without ip range exclusion list", func() {
			mockConfigurator.EXPECT().GetInitContainerImage().Return(containerImage).Times(1)
			mockConfigurator.EXPECT().GetMeshConfig().Return(configv1alpha2.MeshConfig{
				Spec: configv1alpha2.MeshConfigSpec{
					Sidecar: configv1alpha2.SidecarSpec{
						LocalProxyMode: configv1alpha2.LocalProxyModeLocalhost,
					},
				},
			}).Times(1)
			privileged := privilegedFalse
			actual := getInitContainerSpec(containerName, mockConfigurator, nil, nil, nil, nil, privileged, corev1.PullAlways, nil)

			expected := corev1.Container{
				Name:            "-container-name-",
				Image:           "-init-container-image-",
				ImagePullPolicy: corev1.PullAlways,
				Command:         []string{"/bin/sh"},
				Args: []string{
					"-c",
					`iptables-restore --noflush <<EOF
# OSM sidecar interception rules
*nat
:OSM_PROXY_INBOUND - [0:0]
:OSM_PROXY_IN_REDIRECT - [0:0]
:OSM_PROXY_OUTBOUND - [0:0]
:OSM_PROXY_OUT_REDIRECT - [0:0]
-A OSM_PROXY_IN_REDIRECT -p tcp -j REDIRECT --to-port 15003
-A PREROUTING -p tcp -j OSM_PROXY_INBOUND
-A OSM_PROXY_INBOUND -p tcp --dport 15010 -j RETURN
-A OSM_PROXY_INBOUND -p tcp --dport 15901 -j RETURN
-A OSM_PROXY_INBOUND -p tcp --dport 15902 -j RETURN
-A OSM_PROXY_INBOUND -p tcp --dport 15903 -j RETURN
-A OSM_PROXY_INBOUND -p tcp --dport 15904 -j RETURN
-A OSM_PROXY_INBOUND -p tcp -j OSM_PROXY_IN_REDIRECT
-A OSM_PROXY_OUT_REDIRECT -p tcp -j REDIRECT --to-port 15001
-A OSM_PROXY_OUT_REDIRECT -p tcp --dport 15000 -j ACCEPT
-A OUTPUT -p tcp -j OSM_PROXY_OUTBOUND
-A OSM_PROXY_OUTBOUND -o lo ! -d 127.0.0.1/32 -m owner --uid-owner 1500 -j OSM_PROXY_IN_REDIRECT
-A OSM_PROXY_OUTBOUND -o lo -m owner ! --uid-owner 1500 -j RETURN
-A OSM_PROXY_OUTBOUND -m owner --uid-owner 1500 -j RETURN
-A OSM_PROXY_OUTBOUND -d 127.0.0.1/32 -j RETURN
-A OSM_PROXY_OUTBOUND -j OSM_PROXY_OUT_REDIRECT
COMMIT
EOF
`,
				},
				WorkingDir: "",
				Resources:  corev1.ResourceRequirements{},
				SecurityContext: &corev1.SecurityContext{
					Capabilities: &corev1.Capabilities{
						Add: []corev1.Capability{
							"NET_ADMIN",
						},
					},
					Privileged:   &privilegedFalse,
					RunAsNonRoot: &runAsNonRootFalse,
					RunAsUser:    &runAsUserID,
				},
				Env: []corev1.EnvVar{
					{
						Name: "POD_IP",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "v1",
								FieldPath:  "status.podIP",
							},
						},
					},
				},
				Stdin:     false,
				StdinOnce: false,
				TTY:       false,
			}

			Expect(actual).To(Equal(expected))
		})
		It("Sets podIP DNAT rule if set in meshconfig", func() {
			mockConfigurator.EXPECT().GetInitContainerImage().Return(containerImage).Times(1)
			mockConfigurator.EXPECT().GetMeshConfig().Return(configv1alpha2.MeshConfig{
				Spec: configv1alpha2.MeshConfigSpec{
					Sidecar: configv1alpha2.SidecarSpec{
						LocalProxyMode: configv1alpha2.LocalProxyModePodIP,
					},
				},
			}).Times(1)
			privileged := privilegedFalse
			actual := getInitContainerSpec(containerName, mockConfigurator, nil, nil, nil, nil, privileged, corev1.PullAlways, nil)

			expected := corev1.Container{
				Name:            "-container-name-",
				Image:           "-init-container-image-",
				ImagePullPolicy: corev1.PullAlways,
				Command:         []string{"/bin/sh"},
				Args: []string{
					"-c",
					`iptables-restore --noflush <<EOF
# OSM sidecar interception rules
*nat
:OSM_PROXY_INBOUND - [0:0]
:OSM_PROXY_IN_REDIRECT - [0:0]
:OSM_PROXY_OUTBOUND - [0:0]
:OSM_PROXY_OUT_REDIRECT - [0:0]
-A OSM_PROXY_IN_REDIRECT -p tcp -j REDIRECT --to-port 15003
-A PREROUTING -p tcp -j OSM_PROXY_INBOUND
-A OSM_PROXY_INBOUND -p tcp --dport 15010 -j RETURN
-A OSM_PROXY_INBOUND -p tcp --dport 15901 -j RETURN
-A OSM_PROXY_INBOUND -p tcp --dport 15902 -j RETURN
-A OSM_PROXY_INBOUND -p tcp --dport 15903 -j RETURN
-A OSM_PROXY_INBOUND -p tcp --dport 15904 -j RETURN
-A OSM_PROXY_INBOUND -p tcp -j OSM_PROXY_IN_REDIRECT
-A OSM_PROXY_OUT_REDIRECT -p tcp -j REDIRECT --to-port 15001
-A OSM_PROXY_OUT_REDIRECT -p tcp --dport 15000 -j ACCEPT
-A OUTPUT -p tcp -j OSM_PROXY_OUTBOUND
-A OSM_PROXY_OUTBOUND -o lo ! -d 127.0.0.1/32 -m owner --uid-owner 1500 -j OSM_PROXY_IN_REDIRECT
-A OSM_PROXY_OUTBOUND -o lo -m owner ! --uid-owner 1500 -j RETURN
-A OSM_PROXY_OUTBOUND -m owner --uid-owner 1500 -j RETURN
-A OSM_PROXY_OUTBOUND -d 127.0.0.1/32 -j RETURN
-I OUTPUT -p tcp -o lo -d 127.0.0.1/32 -m owner --uid-owner 1500 -j DNAT --to-destination $POD_IP
-A OSM_PROXY_OUTBOUND -j OSM_PROXY_OUT_REDIRECT
COMMIT
EOF
`,
				},
				WorkingDir: "",
				Resources:  corev1.ResourceRequirements{},
				SecurityContext: &corev1.SecurityContext{
					Capabilities: &corev1.Capabilities{
						Add: []corev1.Capability{
							"NET_ADMIN",
						},
					},
					Privileged:   &privilegedFalse,
					RunAsNonRoot: &runAsNonRootFalse,
					RunAsUser:    &runAsUserID,
				},
				Env: []corev1.EnvVar{
					{
						Name: "POD_IP",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "v1",
								FieldPath:  "status.podIP",
							},
						},
					},
				},
				Stdin:     false,
				StdinOnce: false,
				TTY:       false,
			}

			Expect(actual).To(Equal(expected))
		})
	})
})
