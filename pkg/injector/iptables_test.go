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
-I PROXY_INBOUND -p tcp --match multiport --dports 30,40 -j RETURN
-A PROXY_REDIRECT -p tcp -j REDIRECT --to-port 15001
-A PROXY_REDIRECT -p tcp --dport 15000 -j ACCEPT
-A OUTPUT -p tcp -j PROXY_OUTPUT
-A PROXY_OUTPUT -m owner --uid-owner 1500 -j RETURN
-A PROXY_OUTPUT -d 127.0.0.1/32 -j RETURN
-A PROXY_OUTPUT -j PROXY_REDIRECT
-I PROXY_OUTPUT -d 1.1.1.1/32 -j RETURN
-I PROXY_OUTPUT -d 2.2.2.2/32 -j RETURN
-I PROXY_OUTPUT -p tcp --match multiport --dports 10,20 -j RETURN
COMMIT
EOF
`

	assert.Equal(expected, actual)
}
