package ads

import (
	"context"
	"fmt"
	"testing"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	rsrc "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/stream/v3"
	"github.com/envoyproxy/go-control-plane/pkg/test/resource/v3"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// Mocking implementation is referenced from https://github.com/envoyproxy/go-control-plane/blob/v0.9.9/pkg/server/v3/server_test.go
type mockConfigWatcher struct {
	counts     map[string]int
	responses  map[string][]cache.Response
	closeWatch bool
	watches    int
}

func (config *mockConfigWatcher) CreateWatch(req *discovery.DiscoveryRequest) (chan cache.Response, func()) {
	config.counts[req.TypeUrl] = config.counts[req.TypeUrl] + 1
	out := make(chan cache.Response, 1)
	if len(config.responses[req.TypeUrl]) > 0 {
		out <- config.responses[req.TypeUrl][0]
		config.responses[req.TypeUrl] = config.responses[req.TypeUrl][1:]
	} else if config.closeWatch {
		close(out)
	} else {
		config.watches++
		return out, func() {
			// it is ok to close the channel after cancellation and not wait for it to be garbage collected
			close(out)
			config.watches--
		}
	}
	return out, nil
}

func (config *mockConfigWatcher) CreateDeltaWatch(req *discovery.DeltaDiscoveryRequest, st *stream.StreamState) (chan cache.DeltaResponse, func()) {
	return nil, nil
}

func (config *mockConfigWatcher) Fetch(ctx context.Context, req *discovery.DiscoveryRequest) (cache.Response, error) {
	if len(config.responses[req.TypeUrl]) > 0 {
		out := config.responses[req.TypeUrl][0]
		config.responses[req.TypeUrl] = config.responses[req.TypeUrl][1:]
		return out, nil
	}
	return nil, errors.New("missing")
}

func makeMockConfigWatcher() *mockConfigWatcher {
	return &mockConfigWatcher{
		counts: make(map[string]int),
	}
}

type mockStream struct {
	t         *testing.T
	ctx       context.Context
	recv      chan *discovery.DiscoveryRequest
	sent      chan *discovery.DiscoveryResponse
	nonce     int
	sendError bool
	grpc.ServerStream
}

func (stream *mockStream) Context() context.Context {
	return stream.ctx
}

func (stream *mockStream) Send(resp *discovery.DiscoveryResponse) error {
	// check that nonce is monotonically incrementing
	stream.nonce = stream.nonce + 1
	if resp.Nonce != fmt.Sprintf("%d", stream.nonce) {
		stream.t.Errorf("Nonce => got %q, want %d", resp.Nonce, stream.nonce)
	}
	// check that version is set
	if resp.VersionInfo == "" {
		stream.t.Error("VersionInfo => got none, want non-empty")
	}
	// check resources are non-empty
	if len(resp.Resources) == 0 {
		stream.t.Error("Resources => got none, want non-empty")
	}
	// check that type URL matches in resources
	if resp.TypeUrl == "" {
		stream.t.Error("TypeUrl => got none, want non-empty")
	}
	for _, res := range resp.Resources {
		if res.TypeUrl != resp.TypeUrl {
			stream.t.Errorf("TypeUrl => got %q, want %q", res.TypeUrl, resp.TypeUrl)
		}
	}
	stream.sent <- resp
	if stream.sendError {
		return errors.New("send error")
	}
	return nil
}

func (stream *mockStream) Recv() (*discovery.DiscoveryRequest, error) {
	req, more := <-stream.recv
	if !more {
		return nil, errors.New("empty")
	}
	return req, nil
}

func makeMockStream(t *testing.T) *mockStream {
	return &mockStream{
		t:    t,
		ctx:  context.Background(),
		sent: make(chan *discovery.DiscoveryResponse, 10),
		recv: make(chan *discovery.DiscoveryRequest, 10),
	}
}

const (
	clusterName  = "cluster0"
	routeName    = "route0"
	listenerName = "listener0"
	secretName   = "secret0"
	runtimeName  = "runtime0"
)

var (
	node = &core.Node{
		Id:      "test-id",
		Cluster: "test-cluster",
	}
	endpoint   = resource.MakeEndpoint(clusterName, 8080)
	cluster    = resource.MakeCluster(resource.Ads, clusterName)
	route      = resource.MakeRoute(routeName, clusterName)
	listener   = resource.MakeHTTPListener(resource.Ads, listenerName, 80, routeName)
	secret     = resource.MakeSecrets(secretName, "test")[0]
	runtime    = resource.MakeRuntime(runtimeName)
	opaque     = &core.Address{}
	opaqueType = "unknown-type"
)

func makeResponses() map[string][]cache.Response {
	return map[string][]cache.Response{
		rsrc.EndpointType: {
			&cache.RawResponse{
				Version:   "1",
				Resources: []types.ResourceWithTtl{{Resource: endpoint}},
				Request:   &discovery.DiscoveryRequest{TypeUrl: rsrc.EndpointType},
			},
		},
		rsrc.ClusterType: {
			&cache.RawResponse{
				Version:   "2",
				Resources: []types.ResourceWithTtl{{Resource: cluster}},
				Request:   &discovery.DiscoveryRequest{TypeUrl: rsrc.ClusterType},
			},
		},
		rsrc.RouteType: {
			&cache.RawResponse{
				Version:   "3",
				Resources: []types.ResourceWithTtl{{Resource: route}},
				Request:   &discovery.DiscoveryRequest{TypeUrl: rsrc.RouteType},
			},
		},
		rsrc.ListenerType: {
			&cache.RawResponse{
				Version:   "4",
				Resources: []types.ResourceWithTtl{{Resource: listener}},
				Request:   &discovery.DiscoveryRequest{TypeUrl: rsrc.ListenerType},
			},
		},
		rsrc.SecretType: {
			&cache.RawResponse{
				Version:   "5",
				Resources: []types.ResourceWithTtl{{Resource: secret}},
				Request:   &discovery.DiscoveryRequest{TypeUrl: rsrc.SecretType},
			},
		},
		rsrc.RuntimeType: {
			&cache.RawResponse{
				Version:   "6",
				Resources: []types.ResourceWithTtl{{Resource: runtime}},
				Request:   &discovery.DiscoveryRequest{TypeUrl: rsrc.RuntimeType},
			},
		},
		// Pass-through type (xDS does not exist for this type)
		opaqueType: {
			&cache.RawResponse{
				Version:   "7",
				Resources: []types.ResourceWithTtl{{Resource: opaque}},
				Request:   &discovery.DiscoveryRequest{TypeUrl: opaqueType},
			},
		},
	}
}
