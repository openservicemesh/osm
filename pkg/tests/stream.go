package tests

import (
	"context"
	"crypto/x509"

	envoy_service_discovery_v2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"google.golang.org/grpc/peer"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"google.golang.org/grpc/metadata"
)

// XDSServer implements AggregatedDiscoveryService_StreamAggregatedResourcesServer
type XDSServer struct {
	ctx         context.Context
	responses   []*v2.DiscoveryResponse
	requestsCh  chan v2.DiscoveryRequest
	responsesCh chan v2.DiscoveryResponse
}

// NewFakeXDSServer returns a new XDSServer and implements AggregatedDiscoveryService_StreamAggregatedResourcesServer
func NewFakeXDSServer(cert *x509.Certificate, requestsCh chan v2.DiscoveryRequest, responsesCh chan v2.DiscoveryResponse) (envoy_service_discovery_v2.AggregatedDiscoveryService_StreamAggregatedResourcesServer, *[]*v2.DiscoveryResponse) {
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
func (s *XDSServer) Send(r *v2.DiscoveryResponse) error {
	log.Info().Msgf("Fake GRPC XDS sending message: %+v", *r)
	s.responses = append(s.responses, r)
	if s.responsesCh != nil {
		s.responsesCh <- *r
	}
	return nil
}

// Recv implements AggregatedDiscoveryService_StreamAggregatedResourcesServer
func (s *XDSServer) Recv() (*v2.DiscoveryRequest, error) {
	log.Info().Msg("Recv() from Envoy invoked. Waiting on requestsCh.")
	var r v2.DiscoveryRequest
	if s.requestsCh != nil {
		r = <-s.requestsCh
	}
	log.Info().Msgf("Recv() got a DiscoveryRequest from requestsCh: %+v", r)
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
	return s.ctx
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
func (s *XDSServer) SendMsg(m interface{}) error {
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
func (s *XDSServer) RecvMsg(m interface{}) error {
	return nil
}
