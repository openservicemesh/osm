package injector

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestGenerateIptablesCommands(t *testing.T) {
	assert := tassert.New(t)

	outboundIPRangeExclusion := []string{"1.1.1.1/32", "2.2.2.2/32"}
	outboundPortExclusion := []int{10, 20}
	inboundPortExclusion := []int{30, 40}

	actual := generateIptablesCommands(outboundIPRangeExclusion, outboundPortExclusion, inboundPortExclusion)

	expected := `iptables-restore --noflush <<EOF
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
-A OSM_PROXY_INBOUND -p tcp -j OSM_PROXY_IN_REDIRECT
-I OSM_PROXY_INBOUND -p tcp --match multiport --dports 30,40 -j RETURN
-A OSM_PROXY_OUT_REDIRECT -p tcp -j REDIRECT --to-port 15001
-A OSM_PROXY_OUT_REDIRECT -p tcp --dport 15000 -j ACCEPT
-A OUTPUT -p tcp -j OSM_PROXY_OUTBOUND
-A OSM_PROXY_OUTBOUND -o lo ! -d 127.0.0.1/32 -m owner --uid-owner 1500 -j OSM_PROXY_IN_REDIRECT
-A OSM_PROXY_OUTBOUND -o lo -m owner ! --uid-owner 1500 -j RETURN
-A OSM_PROXY_OUTBOUND -m owner --uid-owner 1500 -j RETURN
-A OSM_PROXY_OUTBOUND -d 127.0.0.1/32 -j RETURN
-A OSM_PROXY_OUTBOUND -j OSM_PROXY_OUT_REDIRECT
-I OSM_PROXY_OUTBOUND -d 1.1.1.1/32 -j RETURN
-I OSM_PROXY_OUTBOUND -d 2.2.2.2/32 -j RETURN
-I OSM_PROXY_OUTBOUND -p tcp --match multiport --dports 10,20 -j RETURN
COMMIT
EOF
`

	assert.Equal(expected, actual)
}
