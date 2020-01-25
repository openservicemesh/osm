package utils

import (
	"context"
	"net"
)

const userIPKey int = 0

func GetIPFromContext(ctx context.Context) net.IP {
	userIP, _ := ctx.Value(userIPKey).(net.IP)
	return userIP
}
