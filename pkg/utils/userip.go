package utils

import (
	"context"
	"net"
)

const userIPKey int = 0

// GetIPFromContext obtains the IP address of the caller from the context.
func GetIPFromContext(ctx context.Context) net.IP {
	userIP, _ := ctx.Value(userIPKey).(net.IP)
	return userIP
}
