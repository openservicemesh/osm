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
	privilegedTrue := true

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
					"iptables -t nat -N PROXY_INBOUND && iptables -t nat -N PROXY_IN_REDIRECT && iptables -t nat -N PROXY_OUTPUT && iptables -t nat -N PROXY_REDIRECT && iptables -t nat -A PROXY_REDIRECT -p tcp -j REDIRECT --to-port 15001 && iptables -t nat -A PROXY_REDIRECT -p tcp --dport 15000 -j ACCEPT && iptables -t nat -A OUTPUT -p tcp -j PROXY_OUTPUT && iptables -t nat -A PROXY_OUTPUT -m owner --uid-owner 1500 -j RETURN && iptables -t nat -A PROXY_OUTPUT -d 127.0.0.1/32 -j RETURN && iptables -t nat -A PROXY_OUTPUT -j PROXY_REDIRECT && iptables -t nat -A PROXY_IN_REDIRECT -p tcp -j REDIRECT --to-port 15003 && iptables -t nat -A PREROUTING -p tcp -j PROXY_INBOUND && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15010 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15901 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15902 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15903 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp -j PROXY_IN_REDIRECT",
				},
				WorkingDir: "",
				Resources:  corev1.ResourceRequirements{},
				SecurityContext: &corev1.SecurityContext{
					Capabilities: &corev1.Capabilities{
						Add: []corev1.Capability{
							"NET_ADMIN",
						},
					},
					Privileged: &privilegedFalse,
				},
				Stdin:     false,
				StdinOnce: false,
				TTY:       false,
			}

			Expect(actual).To(Equal(expected))
		})

		It("Creates init container with outbound exclusion list", func() {
			mockConfigurator.EXPECT().GetInitContainerImage().Return(containerImage).Times(1)
			outboundIPRangeExclusionList := []string{"1.1.1.1/32", "10.0.0.10/24"}
			privileged := privilegedFalse
			actual := getInitContainerSpec(containerName, mockConfigurator, outboundIPRangeExclusionList, nil, nil, privileged)

			expected := corev1.Container{
				Name:    "-container-name-",
				Image:   "-init-container-image-",
				Command: []string{"/bin/sh"},
				Args: []string{
					"-c",
					"iptables -t nat -N PROXY_INBOUND && iptables -t nat -N PROXY_IN_REDIRECT && iptables -t nat -N PROXY_OUTPUT && iptables -t nat -N PROXY_REDIRECT && iptables -t nat -A PROXY_REDIRECT -p tcp -j REDIRECT --to-port 15001 && iptables -t nat -A PROXY_REDIRECT -p tcp --dport 15000 -j ACCEPT && iptables -t nat -A OUTPUT -p tcp -j PROXY_OUTPUT && iptables -t nat -A PROXY_OUTPUT -m owner --uid-owner 1500 -j RETURN && iptables -t nat -A PROXY_OUTPUT -d 127.0.0.1/32 -j RETURN && iptables -t nat -A PROXY_OUTPUT -j PROXY_REDIRECT && iptables -t nat -A PROXY_IN_REDIRECT -p tcp -j REDIRECT --to-port 15003 && iptables -t nat -A PREROUTING -p tcp -j PROXY_INBOUND && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15010 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15901 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15902 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15903 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp -j PROXY_IN_REDIRECT && iptables -t nat -I PROXY_OUTPUT -d 1.1.1.1/32 -j RETURN && iptables -t nat -I PROXY_OUTPUT -d 10.0.0.10/24 -j RETURN",
				},
				WorkingDir: "",
				Resources:  corev1.ResourceRequirements{},
				SecurityContext: &corev1.SecurityContext{
					Capabilities: &corev1.Capabilities{
						Add: []corev1.Capability{
							"NET_ADMIN",
						},
					},
					Privileged: &privilegedFalse,
				},
				Stdin:     false,
				StdinOnce: false,
				TTY:       false,
			}

			Expect(actual).To(Equal(expected))
		})

		It("Creates init container with privileged true", func() {
			mockConfigurator.EXPECT().GetInitContainerImage().Return(containerImage).Times(1)
			privileged := privilegedTrue
			actual := getInitContainerSpec(containerName, mockConfigurator, nil, nil, nil, privileged)

			expected := corev1.Container{
				Name:    "-container-name-",
				Image:   "-init-container-image-",
				Command: []string{"/bin/sh"},
				Args: []string{
					"-c",
					"iptables -t nat -N PROXY_INBOUND && iptables -t nat -N PROXY_IN_REDIRECT && iptables -t nat -N PROXY_OUTPUT && iptables -t nat -N PROXY_REDIRECT && iptables -t nat -A PROXY_REDIRECT -p tcp -j REDIRECT --to-port 15001 && iptables -t nat -A PROXY_REDIRECT -p tcp --dport 15000 -j ACCEPT && iptables -t nat -A OUTPUT -p tcp -j PROXY_OUTPUT && iptables -t nat -A PROXY_OUTPUT -m owner --uid-owner 1500 -j RETURN && iptables -t nat -A PROXY_OUTPUT -d 127.0.0.1/32 -j RETURN && iptables -t nat -A PROXY_OUTPUT -j PROXY_REDIRECT && iptables -t nat -A PROXY_IN_REDIRECT -p tcp -j REDIRECT --to-port 15003 && iptables -t nat -A PREROUTING -p tcp -j PROXY_INBOUND && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15010 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15901 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15902 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15903 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp -j PROXY_IN_REDIRECT",
				},
				WorkingDir: "",
				Resources:  corev1.ResourceRequirements{},
				SecurityContext: &corev1.SecurityContext{
					Capabilities: &corev1.Capabilities{
						Add: []corev1.Capability{
							"NET_ADMIN",
						},
					},
					Privileged: &privilegedTrue,
				},
				Stdin:     false,
				StdinOnce: false,
				TTY:       false,
			}

			Expect(actual).To(Equal(expected))
		})

		It("Creates init container without outbound port exclusion list", func() {
			mockConfigurator.EXPECT().GetInitContainerImage().Return(containerImage).Times(1)
			privileged := privilegedFalse
			actual := getInitContainerSpec(containerName, mockConfigurator, nil, nil, nil, privileged)

			expected := corev1.Container{
				Name:    "-container-name-",
				Image:   "-init-container-image-",
				Command: []string{"/bin/sh"},
				Args: []string{
					"-c",
					"iptables -t nat -N PROXY_INBOUND && iptables -t nat -N PROXY_IN_REDIRECT && iptables -t nat -N PROXY_OUTPUT && iptables -t nat -N PROXY_REDIRECT && iptables -t nat -A PROXY_REDIRECT -p tcp -j REDIRECT --to-port 15001 && iptables -t nat -A PROXY_REDIRECT -p tcp --dport 15000 -j ACCEPT && iptables -t nat -A OUTPUT -p tcp -j PROXY_OUTPUT && iptables -t nat -A PROXY_OUTPUT -m owner --uid-owner 1500 -j RETURN && iptables -t nat -A PROXY_OUTPUT -d 127.0.0.1/32 -j RETURN && iptables -t nat -A PROXY_OUTPUT -j PROXY_REDIRECT && iptables -t nat -A PROXY_IN_REDIRECT -p tcp -j REDIRECT --to-port 15003 && iptables -t nat -A PREROUTING -p tcp -j PROXY_INBOUND && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15010 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15901 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15902 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15903 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp -j PROXY_IN_REDIRECT",
				},
				WorkingDir: "",
				Resources:  corev1.ResourceRequirements{},
				SecurityContext: &corev1.SecurityContext{
					Capabilities: &corev1.Capabilities{
						Add: []corev1.Capability{
							"NET_ADMIN",
						},
					},
					Privileged: &privilegedFalse,
				},
				Stdin:     false,
				StdinOnce: false,
				TTY:       false,
			}

			Expect(actual).To(Equal(expected))
		})

		It("init container with outbound port exclusion list", func() {
			mockConfigurator.EXPECT().GetInitContainerImage().Return(containerImage).Times(1)
			outboundPortExclusionList := []int{6060, 7070}
			privileged := privilegedFalse
			actual := getInitContainerSpec(containerName, mockConfigurator, nil, outboundPortExclusionList, nil, privileged)

			expected := corev1.Container{
				Name:    "-container-name-",
				Image:   "-init-container-image-",
				Command: []string{"/bin/sh"},
				Args: []string{
					"-c",
					"iptables -t nat -N PROXY_INBOUND && iptables -t nat -N PROXY_IN_REDIRECT && iptables -t nat -N PROXY_OUTPUT && iptables -t nat -N PROXY_REDIRECT && iptables -t nat -A PROXY_REDIRECT -p tcp -j REDIRECT --to-port 15001 && iptables -t nat -A PROXY_REDIRECT -p tcp --dport 15000 -j ACCEPT && iptables -t nat -A OUTPUT -p tcp -j PROXY_OUTPUT && iptables -t nat -A PROXY_OUTPUT -m owner --uid-owner 1500 -j RETURN && iptables -t nat -A PROXY_OUTPUT -d 127.0.0.1/32 -j RETURN && iptables -t nat -A PROXY_OUTPUT -j PROXY_REDIRECT && iptables -t nat -A PROXY_IN_REDIRECT -p tcp -j REDIRECT --to-port 15003 && iptables -t nat -A PREROUTING -p tcp -j PROXY_INBOUND && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15010 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15901 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15902 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15903 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp -j PROXY_IN_REDIRECT && iptables -t nat -I PROXY_OUTPUT -p tcp --match multiport --dports 6060,7070 -j RETURN",
				},
				WorkingDir: "",
				Resources:  corev1.ResourceRequirements{},
				SecurityContext: &corev1.SecurityContext{
					Capabilities: &corev1.Capabilities{
						Add: []corev1.Capability{
							"NET_ADMIN",
						},
					},
					Privileged: &privilegedFalse,
				},
				Stdin:     false,
				StdinOnce: false,
				TTY:       false,
			}

			Expect(actual).To(Equal(expected))
		})
	})
})
