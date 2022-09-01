package injector

import (
	"testing"

	"github.com/stretchr/testify/assert"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
)

func TestGenerateIptablesCommands(t *testing.T) {
	testCases := []struct {
		name                       string
		proxyMode                  configv1alpha2.LocalProxyMode
		outboundIPRangeExclusions  []string
		outboundIPRangeInclusions  []string
		outboundPortExclusions     []int
		inboundPortExclusions      []int
		networkInterfaceExclusions []string
		expected                   string
	}{
		{
			name: "no exclusions or inclusions",
			expected: `iptables-restore --noflush <<EOF
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
		{
			name:                       "with exclusions and inclusions",
			outboundIPRangeExclusions:  []string{"1.1.1.1/32", "2.2.2.2/32"},
			outboundIPRangeInclusions:  []string{"3.3.3.3/32", "4.4.4.4/32"},
			outboundPortExclusions:     []int{10, 20},
			inboundPortExclusions:      []int{30, 40},
			networkInterfaceExclusions: []string{"eth0", "eth1"},
			expected: `iptables-restore --noflush <<EOF
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
-I OSM_PROXY_INBOUND -i eth0 -j RETURN
-I OSM_PROXY_INBOUND -i eth1 -j RETURN
-I OSM_PROXY_INBOUND -p tcp --match multiport --dports 30,40 -j RETURN
-A OSM_PROXY_OUT_REDIRECT -p tcp -j REDIRECT --to-port 15001
-A OSM_PROXY_OUT_REDIRECT -p tcp --dport 15000 -j ACCEPT
-A OUTPUT -p tcp -j OSM_PROXY_OUTBOUND
-A OSM_PROXY_OUTBOUND -o lo ! -d 127.0.0.1/32 -m owner --uid-owner 1500 -j OSM_PROXY_IN_REDIRECT
-A OSM_PROXY_OUTBOUND -o lo -m owner ! --uid-owner 1500 -j RETURN
-A OSM_PROXY_OUTBOUND -m owner --uid-owner 1500 -j RETURN
-A OSM_PROXY_OUTBOUND -d 127.0.0.1/32 -j RETURN
-A OSM_PROXY_OUTBOUND -o eth0 -j RETURN
-A OSM_PROXY_OUTBOUND -o eth1 -j RETURN
-A OSM_PROXY_OUTBOUND -d 1.1.1.1/32 -j RETURN
-A OSM_PROXY_OUTBOUND -d 2.2.2.2/32 -j RETURN
-A OSM_PROXY_OUTBOUND -p tcp --match multiport --dports 10,20 -j RETURN
-A OSM_PROXY_OUTBOUND -d 3.3.3.3/32 -j OSM_PROXY_OUT_REDIRECT
-A OSM_PROXY_OUTBOUND -d 4.4.4.4/32 -j OSM_PROXY_OUT_REDIRECT
-A OSM_PROXY_OUTBOUND -j RETURN
COMMIT
EOF
`,
		},
		{
			name:      "proxy mode pod ip",
			proxyMode: configv1alpha2.LocalProxyModePodIP,
			expected: `iptables-restore --noflush <<EOF
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			actual := generateIptablesCommands(tc.proxyMode, tc.outboundIPRangeExclusions, tc.outboundIPRangeInclusions, tc.outboundPortExclusions, tc.inboundPortExclusions, tc.networkInterfaceExclusions)
			a.Equal(tc.expected, actual)
		})
	}
}
