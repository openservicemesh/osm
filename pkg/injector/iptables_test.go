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

	expected := []string{
		"iptables -t nat -N PROXY_INBOUND",
		"iptables -t nat -N PROXY_IN_REDIRECT",
		"iptables -t nat -N PROXY_OUTPUT",
		"iptables -t nat -N PROXY_REDIRECT",
		"iptables -t nat -A PROXY_REDIRECT -p tcp -j REDIRECT --to-port 15001",
		"iptables -t nat -A PROXY_REDIRECT -p tcp --dport 15000 -j ACCEPT",
		"iptables -t nat -A OUTPUT -p tcp -j PROXY_OUTPUT",
		"iptables -t nat -A PROXY_OUTPUT -m owner --uid-owner 1500 -j RETURN",
		"iptables -t nat -A PROXY_OUTPUT -d 127.0.0.1/32 -j RETURN",
		"iptables -t nat -A PROXY_OUTPUT -j PROXY_REDIRECT",
		"iptables -t nat -A PROXY_IN_REDIRECT -p tcp -j REDIRECT --to-port 15003",
		"iptables -t nat -A PREROUTING -p tcp -j PROXY_INBOUND",
		"iptables -t nat -A PROXY_INBOUND -p tcp --dport 15010 -j RETURN",
		"iptables -t nat -A PROXY_INBOUND -p tcp --dport 15901 -j RETURN",
		"iptables -t nat -A PROXY_INBOUND -p tcp --dport 15902 -j RETURN",
		"iptables -t nat -A PROXY_INBOUND -p tcp --dport 15903 -j RETURN",
		"iptables -t nat -A PROXY_INBOUND -p tcp -j PROXY_IN_REDIRECT",
		"iptables -t nat -I PROXY_OUTPUT -d 1.1.1.1/32 -j RETURN",
		"iptables -t nat -I PROXY_OUTPUT -d 2.2.2.2/32 -j RETURN",
		"iptables -t nat -I PROXY_OUTPUT -p tcp --match multiport --dports 10,20 -j RETURN",
		"iptables -t nat -I PROXY_INBOUND -p tcp --match multiport --dports 30,40 -j RETURN",
	}

	assert.ElementsMatch(expected, actual)
}
