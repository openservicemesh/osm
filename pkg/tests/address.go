package tests

import "net"

type netAddr struct {
	address string
}

// Network implements net.Addr interface
func (a *netAddr) Network() string {
	return "mockNetwork"
}

func (a *netAddr) String() string {
	return a.address
}

// NewMockAddress creates a new net.Addr
func NewMockAddress(address string) net.Addr {
	return &netAddr{
		address: address,
	}
}
