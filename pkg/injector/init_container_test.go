package injector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	corev1 "k8s.io/api/core/v1"

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
			privileged := privilegedFalse
			actual := getInitContainerSpec(containerName, mockConfigurator, nil, nil, nil, privileged)

			expected := corev1.Container{
				Name:    "-container-name-",
				Image:   "-init-container-image-",
				Command: []string{"/bin/sh"},
				Args: []string{
					"-c",
					`iptables-restore --noflush <<EOF
# OSM sidecar interception rules
*nat
:PROXY_INBOUND - [0:0]
:PROXY_IN_REDIRECT - [0:0]
:PROXY_OUTPUT - [0:0]
:PROXY_REDIRECT - [0:0]
-A PROXY_IN_REDIRECT -p tcp -j REDIRECT --to-port 15003
-A PREROUTING -p tcp -j PROXY_INBOUND
-A PROXY_INBOUND -p tcp --dport 15010 -j RETURN
-A PROXY_INBOUND -p tcp --dport 15901 -j RETURN
-A PROXY_INBOUND -p tcp --dport 15902 -j RETURN
-A PROXY_INBOUND -p tcp --dport 15903 -j RETURN
-A PROXY_INBOUND -p tcp -j PROXY_IN_REDIRECT
-A PROXY_REDIRECT -p tcp -j REDIRECT --to-port 15001
-A PROXY_REDIRECT -p tcp --dport 15000 -j ACCEPT
-A OUTPUT -p tcp -j PROXY_OUTPUT
-A PROXY_OUTPUT -m owner --uid-owner 1500 -j RETURN
-A PROXY_OUTPUT -d 127.0.0.1/32 -j RETURN
-A PROXY_OUTPUT -j PROXY_REDIRECT
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
				Stdin:     false,
				StdinOnce: false,
				TTY:       false,
			}

			Expect(actual).To(Equal(expected))
		})
	})
})
