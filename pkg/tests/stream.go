package tests

import (
	"context"
	"crypto/x509"

	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

// XDSServer implements AggregatedDiscoveryService_StreamAggregatedResourcesServer
type XDSServer struct {
	ctx         context.Context
	responses   []*xds_discovery.DiscoveryResponse
	requestsCh  chan xds_discovery.DiscoveryRequest
	responsesCh chan xds_discovery.DiscoveryResponse
}

// NewFakeXDSServer returns a new XDSServer and implements AggregatedDiscoveryService_StreamAggregatedResourcesServer
func NewFakeXDSServer(cert *x509.Certificate, requestsCh chan xds_discovery.DiscoveryRequest, responsesCh chan xds_discovery.DiscoveryResponse) (xds_discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer, *[]*xds_discovery.DiscoveryResponse) {
	peerKey := peer.Peer{
		Addr:     NewMockAddress("9.8.7.6"),
		AuthInfo: NewMockAuthInfo(cert),
	}
	server := XDSServer{
		ctx:         peer.NewContext(context.TODO(), &peerKey),
		requestsCh:  requestsCh,
		responsesCh: responsesCh,
	}
	return &server, &server.responses
}

// Send implements AggregatedDiscoveryService_StreamAggregatedResourcesServer
func (s *XDSServer) Send(r *xds_discovery.DiscoveryResponse) error {
	s.responses = append(s.responses, r)
	return nil
}

// Recv implements AggregatedDiscoveryService_StreamAggregatedResourcesServer
func (s *XDSServer) Recv() (*xds_discovery.DiscoveryRequest, error) {
	r := xds_discovery.DiscoveryRequest{
		VersionInfo:   "",
		Node:          nil,
		ResourceNames: nil,
		TypeUrl:       "",
		ResponseNonce: "",
		ErrorDetail:   nil,
	}
	return &r, nil
}

// SetHeader sets the header metadata. It may be called multiple times.
// When call multiple times, all the provided metadata will be merged.
// All the metadata will be sent out when one of the following happens:
//  - ServerStream.SendHeader() is called;
//  - The first response is sent out;
//  - An RPC status is sent out (error or success).
func (s *XDSServer) SetHeader(metadata.MD) error {
	return nil
}

// SendHeader sends the header metadata.
// The provided md and headers set by SetHeader() will be sent.
// It fails if called multiple times.
func (s *XDSServer) SendHeader(metadata.MD) error {
	return nil
}

// SetTrailer sets the trailer metadata which will be sent with the RPC status.
// When called more than once, all the provided metadata will be merged.
func (s *XDSServer) SetTrailer(metadata.MD) {
}

// Context returns the context for this stream.
func (s *XDSServer) Context() context.Context {
	return nil
}

// SendMsg sends a message. On error, SendMsg aborts the stream and the
// error is returned directly.
//
// SendMsg blocks until:
//   - There is sufficient flow control to schedule m with the transport, or
//   - The stream is done, or
//   - The stream breaks.
//
// SendMsg does not wait until the message is received by the client. An
// untimely stream closure may result in lost messages.
//
// It is safe to have a goroutine calling SendMsg and another goroutine
// calling RecvMsg on the same stream at the same time, but it is not safe
// to call SendMsg on the same stream in different goroutines.
func (s *XDSServer) SendMsg(_ interface{}) error {
	return nil
}

// RecvMsg blocks until it receives a message into m or the stream is
// done. It returns io.EOF when the client has performed a CloseSend. On
// any non-EOF error, the stream is aborted and the error contains the
// RPC status.
//
// It is safe to have a goroutine calling SendMsg and another goroutine
// calling RecvMsg on the same stream at the same time, but it is not
// safe to call RecvMsg on the same stream in different goroutines.
func (s *XDSServer) RecvMsg(_ interface{}) error {
	return nil
}
