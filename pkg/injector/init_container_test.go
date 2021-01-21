package injector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

var _ = Describe("Test volume functions", func() {
	Context("Test getInitContainerSpec", func() {
		It("creates volume spec", func() {
			actual := getInitContainerSpec("-container-name-", "-init-container-image-")
			expected := v1.Container{
				Name:    "-container-name-",
				Image:   "-init-container-image-",
				Command: []string{"/bin/sh"},
				Args: []string{
					"-c",
					"iptables -t nat -N PROXY_REDIRECT && iptables -t nat -A PROXY_REDIRECT -p tcp -j REDIRECT --to-port 15001 && iptables -t nat -A PROXY_REDIRECT -p tcp --dport 15000 -j ACCEPT && iptables -t nat -N PROXY_IN_REDIRECT && iptables -t nat -A PROXY_IN_REDIRECT -p tcp -j REDIRECT --to-port 15003 && iptables -t nat -N PROXY_INBOUND && iptables -t nat -A PREROUTING -p tcp -j PROXY_INBOUND && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15010 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15901 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15902 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15903 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp -j PROXY_IN_REDIRECT && iptables -t nat -N PROXY_OUTPUT && iptables -t nat -A OUTPUT -p tcp -j PROXY_OUTPUT && iptables -t nat -A PROXY_OUTPUT -m owner --uid-owner 1337 -j RETURN && iptables -t nat -A PROXY_OUTPUT -d 127.0.0.1/32 -j RETURN && iptables -t nat -A PROXY_OUTPUT -j PROXY_REDIRECT",
				},
				WorkingDir: "",
				Resources:  v1.ResourceRequirements{},
				SecurityContext: &v1.SecurityContext{
					Capabilities: &v1.Capabilities{
						Add: []v1.Capability{
							"NET_ADMIN",
						},
					},
					Privileged: nil,
				},
				Stdin:     false,
				StdinOnce: false,
				TTY:       false,
			}
			Expect(actual).To(Equal(expected))
		})
	})
})
