package injector

import (
	"fmt"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestGetInitContainerSpec(t *testing.T) {
	assert := tassert.New(t)

	containerName := "-container-name-"
	containerImage := "-init-container-image-"

	testCases := []struct {
		name                         string
		outboundIPRangeExclusionList []string

		expectedSpec v1.Container
	}{
		{
			name:                         "init container without outbound exclusion list",
			outboundIPRangeExclusionList: nil,

			expectedSpec: v1.Container{
				Name:    "-container-name-",
				Image:   "-init-container-image-",
				Command: []string{"/bin/sh"},
				Args: []string{
					"-c",
					"iptables -t nat -N PROXY_INBOUND && iptables -t nat -N PROXY_IN_REDIRECT && iptables -t nat -N PROXY_OUTPUT && iptables -t nat -N PROXY_REDIRECT && iptables -t nat -A PROXY_REDIRECT -p tcp -j REDIRECT --to-port 15001 && iptables -t nat -A PROXY_REDIRECT -p tcp --dport 15000 -j ACCEPT && iptables -t nat -A OUTPUT -p tcp -j PROXY_OUTPUT && iptables -t nat -A PROXY_OUTPUT -m owner --uid-owner 1337 -j RETURN && iptables -t nat -A PROXY_OUTPUT -d 127.0.0.1/32 -j RETURN && iptables -t nat -A PROXY_OUTPUT -j PROXY_REDIRECT && iptables -t nat -A PROXY_IN_REDIRECT -p tcp -j REDIRECT --to-port 15003 && iptables -t nat -A PREROUTING -p tcp -j PROXY_INBOUND && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15010 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15901 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15902 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15903 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp -j PROXY_IN_REDIRECT",
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
			},
		},

		{
			name:                         "init container with outbound exclusion list",
			outboundIPRangeExclusionList: []string{"1.1.1.1/32", "10.0.0.10/24"},

			expectedSpec: v1.Container{
				Name:    "-container-name-",
				Image:   "-init-container-image-",
				Command: []string{"/bin/sh"},
				Args: []string{
					"-c",
					"iptables -t nat -N PROXY_INBOUND && iptables -t nat -N PROXY_IN_REDIRECT && iptables -t nat -N PROXY_OUTPUT && iptables -t nat -N PROXY_REDIRECT && iptables -t nat -A PROXY_REDIRECT -p tcp -j REDIRECT --to-port 15001 && iptables -t nat -A PROXY_REDIRECT -p tcp --dport 15000 -j ACCEPT && iptables -t nat -A OUTPUT -p tcp -j PROXY_OUTPUT && iptables -t nat -A PROXY_OUTPUT -m owner --uid-owner 1337 -j RETURN && iptables -t nat -A PROXY_OUTPUT -d 127.0.0.1/32 -j RETURN && iptables -t nat -A PROXY_OUTPUT -j PROXY_REDIRECT && iptables -t nat -A PROXY_IN_REDIRECT -p tcp -j REDIRECT --to-port 15003 && iptables -t nat -A PREROUTING -p tcp -j PROXY_INBOUND && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15010 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15901 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15902 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15903 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp -j PROXY_IN_REDIRECT && iptables -t nat -I PROXY_OUTPUT -d 1.1.1.1/32 -j RETURN && iptables -t nat -I PROXY_OUTPUT -d 10.0.0.10/24 -j RETURN",
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
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			actual := getInitContainerSpec(containerName, containerImage, tc.outboundIPRangeExclusionList)
			assert.Equal(tc.expectedSpec, actual)
		})
	}
}
