package registry

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/messaging"
)

func TestReleaseCertificateHandler(t *testing.T) {
	podUID := uuid.New().String()
	proxyCN := certificate.CommonName(fmt.Sprintf("%s.sidecar.foo.bar", podUID))

	testCases := []struct {
		name       string
		eventFunc  func(*messaging.Broker)
		assertFunc func(*assert.Assertions, certificate.Manager)
	}{
		{
			name: "The certificate is released when the corresponding pod is deleted",
			eventFunc: func(m *messaging.Broker) {
				m.GetKubeEventPubSub().Pub(events.PubSubMessage{
					Kind:   announcements.PodDeleted,
					NewObj: nil,
					OldObj: &v1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							UID: types.UID(podUID),
						},
					},
				}, announcements.PodDeleted.String())
			},
			assertFunc: func(a *assert.Assertions, cm certificate.Manager) {
				a.Eventually(func() bool {
					cert, err := cm.GetCertificate(proxyCN)
					return err != nil && cert == nil
				}, 2*time.Second, 500*time.Millisecond)
			},
		},
		{
			name: "The certificate is not released when an unrelated pod is deleted",
			eventFunc: func(m *messaging.Broker) {
				m.GetKubeEventPubSub().Pub(events.PubSubMessage{
					Kind:   announcements.PodDeleted,
					NewObj: nil,
					OldObj: &v1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							UID: types.UID(uuid.New().String()), // Pod UUID does not match cert
						},
					},
				}, announcements.PodDeleted.String())
			},
			assertFunc: func(a *assert.Assertions, cm certificate.Manager) {
				// Give enough time for the cert to be removed
				// and only then verify that the cert still exists.
				// This delay is important because even when the cert is
				// removed, it happens asynchronously.
				time.Sleep(2 * time.Second)
				cert, err := cm.GetCertificate(proxyCN)
				a.Nil(err)
				a.NotNil(cert)
			},
		},
		{
			name: "The certificate is not released when an event other than PodDeleted is received",
			eventFunc: func(m *messaging.Broker) {
				m.GetKubeEventPubSub().Pub(events.PubSubMessage{
					Kind:   announcements.PodAdded,
					NewObj: nil,
					OldObj: &v1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							UID: types.UID(podUID),
						},
					},
				}, announcements.PodAdded.String())
			},
			assertFunc: func(a *assert.Assertions, cm certificate.Manager) {
				// Give enough time for the cert to be removed
				// and only then verify that the cert still exists.
				// This delay is important because even when the cert is
				// removed, it happens asynchronously.
				time.Sleep(2 * time.Second)
				cert, err := cm.GetCertificate(proxyCN)
				a.Nil(err)
				a.NotNil(cert)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			stop := make(chan struct{})
			defer close(stop)

			msgBroker := messaging.NewBroker(stop)
			proxyRegistry := NewProxyRegistry(nil, msgBroker)
			certManager := tresor.NewFake(msgBroker)

			_, err := certManager.IssueCertificate(proxyCN, 1*time.Hour)
			a.Nil(err)
			cert, err := certManager.GetCertificate(proxyCN)
			a.Nil(err)
			a.NotNil(cert)

			proxy, err := envoy.NewProxy(proxyCN, "-cert-serial-number-", nil)
			a.Nil(err)

			proxy.PodMetadata = &envoy.PodMetadata{
				UID: podUID,
			}

			proxyRegistry.RegisterProxy(proxy)

			go proxyRegistry.ReleaseCertificateHandler(certManager, stop)
			// Subscription should happen before an event is published by the test, so
			// add a delay before the test triggers events
			time.Sleep(500 * time.Millisecond)

			tc.eventFunc(msgBroker)
			tc.assertFunc(a, certManager)
		})
	}
}
