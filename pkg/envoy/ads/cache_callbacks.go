package ads

import (
	"context"
	"fmt"
	"sync"
	"time"

	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/metricsstore"
	"github.com/openservicemesh/osm/pkg/utils"
)

// OnStreamOpen is called on stream open
func (s *Server) OnStreamOpen(ctx context.Context, streamID int64, typ string) error {
	log.Debug().Msgf("OnStreamOpen id: %d typ: %s", streamID, typ)
	// When a new Envoy proxy connects, ValidateClient would ensure that it has a valid certificate,
	// and the Subject CN is in the allowedCommonNames set.
	certCommonName, certSerialNumber, err := utils.ValidateClient(ctx)
	if err != nil {
		return fmt.Errorf("Could not start Aggregated Discovery Service gRPC stream for newly connected Envoy proxy: %w", err)
	}

	// If maxDataPlaneConnections is enabled i.e. not 0, then check that the number of Envoy connections is less than maxDataPlaneConnections
	if s.catalog.GetMeshConfig().Spec.Sidecar.MaxDataPlaneConnections > 0 && s.proxyRegistry.GetConnectedProxyCount() >= s.catalog.GetMeshConfig().Spec.Sidecar.MaxDataPlaneConnections {
		metricsstore.DefaultMetricsStore.ProxyMaxConnectionsRejected.Inc()
		return errTooManyConnections
	}

	log.Trace().Msgf("Envoy with certificate SerialNumber=%s connected", certSerialNumber)
	metricsstore.DefaultMetricsStore.ProxyConnectCount.Inc()

	kind, uuid, si, err := getCertificateCommonNameMeta(certCommonName)
	if err != nil {
		return fmt.Errorf("error parsing certificate common name %s: %w", certCommonName, err)
	}

	proxy := envoy.NewProxy(kind, uuid, si, utils.GetIPFromContext(ctx), streamID)

	if err := s.catalog.VerifyProxy(proxy); err != nil {
		return err
	}

	s.proxyRegistry.RegisterProxy(proxy)
	go func() {
		// Register for proxy config updates broadcasted by the message broker
		proxyUpdatePubSub := s.msgBroker.GetProxyUpdatePubSub()
		proxyUpdateChan := proxyUpdatePubSub.Sub(messaging.ProxyUpdateTopic, messaging.GetPubSubTopicForProxyUUID(proxy.UUID.String()))
		defer s.msgBroker.Unsub(proxyUpdatePubSub, proxyUpdateChan)

		certRotations, unsubRotations := s.certManager.SubscribeRotations(proxy.Identity.String())
		defer unsubRotations()

		// schedule one update for this proxy initially.
		s.scheduleUpdate(proxy)
		for {
			select {
			case <-proxyUpdateChan:
				log.Debug().Str("proxy", proxy.String()).Msg("Broadcast update received")
				s.scheduleUpdate(proxy)
			case <-certRotations:
				log.Debug().Str("proxy", proxy.String()).Msg("Certificate has been updated for proxy")
				s.scheduleUpdate(proxy)
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (s *Server) scheduleUpdate(proxy *envoy.Proxy) {
	var wg sync.WaitGroup
	wg.Add(1)
	s.workqueues.AddJob(
		func() {
			t := time.Now()
			log.Debug().Msgf("Starting update for proxy %s", proxy.String())

			if err := s.update(proxy); err != nil {
				log.Error().Err(err).Str("proxy", proxy.String()).Msg("Error generating resources for proxy")
			}
			log.Debug().Msgf("Update for proxy %s took took %v", proxy.String(), time.Since(t))
			wg.Done()
		})
	wg.Wait()
}

func (s *Server) update(proxy *envoy.Proxy) error {
	resources, err := s.GenerateResources(proxy)
	if err != nil {
		return err
	}
	if err := s.ServeResources(proxy, resources); err != nil {
		return err
	}
	log.Debug().Msgf("successfully updated resources for proxy %s", proxy.String())
	return nil
}

// OnStreamClosed is called on stream closed
func (s *Server) OnStreamClosed(streamID int64) {
	log.Debug().Msgf("OnStreamClosed id: %d", streamID)
	s.proxyRegistry.UnregisterProxy(streamID)

	metricsstore.DefaultMetricsStore.ProxyConnectCount.Dec()
}

// OnStreamRequest is called when a request happens on an open connection
func (s *Server) OnStreamRequest(streamID int64, req *discovery.DiscoveryRequest) error {
	log.Debug().Msgf("OnStreamRequest node: %s, type: %s, v: %s, nonce: %s, resNames: %s", req.Node.Id, req.TypeUrl, req.VersionInfo, req.ResponseNonce, req.ResourceNames)

	proxy := s.proxyRegistry.GetConnectedProxy(streamID)
	if proxy != nil {
		metricsstore.DefaultMetricsStore.ProxyXDSRequestCount.WithLabelValues(proxy.UUID.String(), proxy.Identity.String(), req.TypeUrl).Inc()
	}

	return nil
}

// OnStreamResponse is called when a response is being sent to a request
func (s *Server) OnStreamResponse(_ context.Context, streamID int64, req *discovery.DiscoveryRequest, resp *discovery.DiscoveryResponse) {
	log.Debug().Msgf("OnStreamResponse RESP: %d type: %s, v: %s, nonce: %s, NumResources: %d", streamID, resp.TypeUrl, resp.VersionInfo, resp.Nonce, len(resp.Resources))
}

// --- Fetch request types. Callback interfaces still requires these to be defined

// OnFetchRequest is called when a fetch request is received
func (s *Server) OnFetchRequest(_ context.Context, req *discovery.DiscoveryRequest) error {
	// Unimplemented
	return errUnsuportedXDSRequest
}

// OnFetchResponse is called when a fetch request is being responded to
func (s *Server) OnFetchResponse(req *discovery.DiscoveryRequest, resp *discovery.DiscoveryResponse) {
	// Unimplemented
}

// --- Delta stream types below. Callback interfaces still requires these to be defined

// OnDeltaStreamOpen is called when a Delta stream is being opened
func (s *Server) OnDeltaStreamOpen(_ context.Context, id int64, typ string) error {
	// Unimplemented
	return errUnsuportedXDSRequest
}

// OnDeltaStreamClosed is called when a Delta stream is being closed
func (s *Server) OnDeltaStreamClosed(id int64) {
	// Unimplemented
}

// OnStreamDeltaRequest is called when a Delta request comes on an open Delta stream
func (s *Server) OnStreamDeltaRequest(a int64, req *discovery.DeltaDiscoveryRequest) error {
	// Unimplemented
	return errUnsuportedXDSRequest
}

// OnStreamDeltaResponse is called when a Delta request is getting responded to
func (s *Server) OnStreamDeltaResponse(a int64, req *discovery.DeltaDiscoveryRequest, resp *discovery.DeltaDiscoveryResponse) {
	// Unimplemented
}
