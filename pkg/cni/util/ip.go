package util

import (
	"fmt"
	"net"
	"unsafe"
)

// IP2Pointer returns the pointer of a ip string
func IP2Pointer(ipstr string) (unsafe.Pointer, error) {
	ip := net.ParseIP(ipstr)
	if ip == nil {
		return nil, fmt.Errorf("error parse ip: %s", ipstr)
	}
	if ip.To4() != nil {
		// ipv4, we need to clear the bytes
		for i := 0; i < 12; i++ {
			ip[i] = 0
		}
	}
	//#nosec G103
	return unsafe.Pointer(&ip[0]), nil
}
