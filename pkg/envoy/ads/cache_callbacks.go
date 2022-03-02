package ads

import (
	"context"

	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
)

// Callbacks is an implementation of xDS server callbacks required by go-control-plane
// These are put in case we want to add any logic to specific parts of the xDS server proto handling implementation.
// Though mandatory to be provided, they are not required to do anything, but can certainly help to understand,
// debug and instrument additional functionality on top of the cache.
// Sample implementation from https://github.com/envoyproxy/go-control-plane/blob/main/docs/cache/Server.md
type Callbacks struct {
	// empty
}

// OnStreamOpen is called on stream open
func (cb *Callbacks) OnStreamOpen(_ context.Context, id int64, typ string) error {
	// TODO: Validate context
	log.Debug().Msgf("OnStreamOpen id: %d typ: %s", id, typ)
	return nil
}

// OnStreamClosed is called on stream closed
func (cb *Callbacks) OnStreamClosed(id int64) {
	log.Debug().Msgf("OnStreamClosed id: %d", id)
}

// OnStreamRequest is called when a request happens on an open string
func (cb *Callbacks) OnStreamRequest(a int64, req *discovery.DiscoveryRequest) error {
	log.Debug().Msgf("OnStreamRequest node: %s, type: %s, v: %s, nonce: %s, resNames: %s", req.Node.Id, req.TypeUrl, req.VersionInfo, req.ResponseNonce, req.ResourceNames)
	return nil
}

// OnStreamResponse is called when a response is being sent to a request
func (cb *Callbacks) OnStreamResponse(_ context.Context, aa int64, req *discovery.DiscoveryRequest, resp *discovery.DiscoveryResponse) {
	log.Debug().Msgf("OnStreamResponse REQ: %s, type: %s, v: %s, nonce: %s, resNames: %s", req.Node.Id, req.TypeUrl, req.VersionInfo, req.ResponseNonce, req.ResourceNames)
	log.Debug().Msgf("OnStreamResponse RESP: type: %s, v: %s, nonce: %s, NumResources: %d", resp.TypeUrl, resp.VersionInfo, resp.Nonce, len(resp.Resources))
}

// --- Fetch request types. Callback interfaces still requires these to be defined

// OnFetchRequest is called when a fetch request is received
func (cb *Callbacks) OnFetchRequest(_ context.Context, req *discovery.DiscoveryRequest) error {
	// Unimplemented
	return errUnsuportedXDSRequest
}

// OnFetchResponse is called when a fetch request is being responded to
func (cb *Callbacks) OnFetchResponse(req *discovery.DiscoveryRequest, resp *discovery.DiscoveryResponse) {
	// Unimplemented
}

// --- Delta stream types below. Callback interfaces still requires these to be defined

// OnDeltaStreamOpen is called when a Delta stream is being opened
func (cb *Callbacks) OnDeltaStreamOpen(_ context.Context, id int64, typ string) error {
	// Unimplemented
	return errUnsuportedXDSRequest
}

// OnDeltaStreamClosed is called when a Delta stream is being closed
func (cb *Callbacks) OnDeltaStreamClosed(id int64) {
	// Unimplemented
}

// OnStreamDeltaRequest is called when a Delta request comes on an open Delta stream
func (cb *Callbacks) OnStreamDeltaRequest(a int64, req *discovery.DeltaDiscoveryRequest) error {
	// Unimplemented
	return errUnsuportedXDSRequest
}

// OnStreamDeltaResponse is called when a Delta request is getting responded to
func (cb *Callbacks) OnStreamDeltaResponse(a int64, req *discovery.DeltaDiscoveryRequest, resp *discovery.DeltaDiscoveryResponse) {
	// Unimplemented
}
