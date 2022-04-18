package injector

import (
	"testing"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/stretchr/testify/assert"
)

func TestGenerateIptablesCommands(t *testing.T) {
	testCases := []struct {
		name                      string
		configurator              configurator.Configurator
		outboundIPRangeExclusions []string
		outboundIPRangeInclusions []string
		outboundPortExclusions    []int
		inboundPortExclusions     []int
		expected                  string
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
			name:                      "with exclusions and inclusions",
			outboundIPRangeExclusions: []string{"1.1.1.1/32", "2.2.2.2/32"},
			outboundIPRangeInclusions: []string{"3.3.3.3/32", "4.4.4.4/32"},
			outboundPortExclusions:    []int{10, 20},
			inboundPortExclusions:     []int{30, 40},
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
-I OSM_PROXY_INBOUND -p tcp --match multiport --dports 30,40 -j RETURN
-A OSM_PROXY_OUT_REDIRECT -p tcp -j REDIRECT --to-port 15001
-A OSM_PROXY_OUT_REDIRECT -p tcp --dport 15000 -j ACCEPT
-A OUTPUT -p tcp -j OSM_PROXY_OUTBOUND
-A OSM_PROXY_OUTBOUND -o lo ! -d 127.0.0.1/32 -m owner --uid-owner 1500 -j OSM_PROXY_IN_REDIRECT
-A OSM_PROXY_OUTBOUND -o lo -m owner ! --uid-owner 1500 -j RETURN
-A OSM_PROXY_OUTBOUND -m owner --uid-owner 1500 -j RETURN
-A OSM_PROXY_OUTBOUND -d 127.0.0.1/32 -j RETURN
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)
			actual := generateIptablesCommands(tc.configurator, tc.outboundIPRangeExclusions, tc.outboundIPRangeInclusions, tc.outboundPortExclusions, tc.inboundPortExclusions)
			a.Equal(tc.expected, actual)
		})
	}
}
