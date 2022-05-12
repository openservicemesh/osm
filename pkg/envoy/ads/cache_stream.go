package ads

import (
	"context"
	"fmt"
	"net"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/errcode"
)

// Routine which fulfills listening to proxy broadcasts
func (s *Server) broadcastListener() {
	// Register for proxy config updates broadcasted by the message broker
	proxyUpdatePubSub := s.msgBroker.GetProxyUpdatePubSub()
	proxyUpdateChan := proxyUpdatePubSub.Sub(announcements.ProxyUpdate.String())
	defer s.msgBroker.Unsub(proxyUpdatePubSub, proxyUpdateChan)

	for {
		<-proxyUpdateChan
		s.allPodUpdater()
	}
}

func (s *Server) allPodUpdater() {
	allpods := s.kubecontroller.ListPods()

	for _, pod := range allpods {
		proxy, err := GetProxyFromPod(pod)
		if err != nil {
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrGettingProxyFromPod)).
				Msgf("Could not get proxy from pod %s/%s", pod.Namespace, pod.Name)
			continue
		}

		// Queue update for this proxy/pod
		job := proxyResponseJob{
			proxy:     proxy,
			adsStream: nil, // Since it goes in the cache, stream is not needed
			request:   nil, // No request is used, as we fill all verticals
			xdsServer: s,
			typeURIs:  envoy.XDSResponseOrder,
			done:      make(chan struct{}),
		}
		s.workqueues.AddJob(&job)
	}
}

// GetProxyFromPod infers and creates a Proxy data structure from a Pod.
// This is a temporary workaround as proxy is required and expected in any vertical call to XDS,
// however snapshotcache has no need to provide visibility on proxies whatsoever.
// All verticals use the proxy structure to infer the pod later, so the actual only mandatory
// data for the verticals to be functional is the common name, which links proxy <-> pod
func GetProxyFromPod(pod *v1.Pod) (*envoy.Proxy, error) {
	var serviceAccount string
	var namespace string

	uuidString, uuidFound := pod.Labels[constants.EnvoyUniqueIDLabelName]
	if !uuidFound {
		return nil, errors.Errorf("UUID not found for pod %s/%s, not a mesh pod", pod.Namespace, pod.Name)
	}
	proxyUUID, err := uuid.Parse(uuidString)
	if err != nil {
		return nil, errors.Errorf("Could not parse UUID label into UUID type (%s): %v", uuidString, err)
	}

	serviceAccount = pod.Spec.ServiceAccountName
	namespace = pod.Namespace

	// construct CN for this pod/proxy
	// TODO: Infer proxy type from Pod
	commonName := envoy.NewXDSCertCommonName(proxyUUID, envoy.KindSidecar, serviceAccount, namespace)
	tempProxy, err := envoy.NewProxy(certificate.CommonName(commonName), "NoSerial", &net.IPAddr{IP: net.IPv4zero})

	return tempProxy, err
}

// RecordFullSnapshot stores a group of resources as a new Snapshot with a new version in the cache.
// It also runs a consistency check on the snapshot (will warn if there are missing resources referenced in
// the snapshot)
func (s *Server) RecordFullSnapshot(proxy *envoy.Proxy, snapshotResources map[string][]types.Resource) error {
	s.configVerMutex.Lock()
	s.configVersion[proxy.GetCertificateCommonName().String()]++
	s.configVerMutex.Unlock()

	snapshot, err := cache.NewSnapshot(
		fmt.Sprintf("%d", s.configVersion[proxy.GetCertificateCommonName().String()]),
		snapshotResources,
	)
	if err != nil {
		return err
	}

	if err := snapshot.Consistent(); err != nil {
		log.Warn().Err(err).Str("proxy", proxy.String()).Msgf("Snapshot for proxy not consistent")
	}

	return s.ch.SetSnapshot(context.TODO(), proxy.GetCertificateCommonName().String(), snapshot)
}
