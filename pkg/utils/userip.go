package utils

import (
	"context"
	"net"

	"google.golang.org/grpc/peer"
)

// GetIPFromContext obtains the IP address of the caller from the context.
func GetIPFromContext(ctx context.Context) net.Addr {
	if clientPeer, ok := peer.FromContext(ctx); ok {
		return clientPeer.Addr
	}
	return nil
}
