package webhook

import (
	"context"
	"testing"
	"time"

	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/certificate"
	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/messaging"
)

func TestNewServer(t *testing.T) {
	assert := tassert.New(t)
	cm := tresorFake.NewFake(nil, time.Hour)

	stop := make(chan struct{})
	defer close(stop)
	broker := messaging.NewBroker(stop)

	ctx, cancel := context.WithCancel(context.Background())

	var count int

	s, err := NewServer("my-webhook", "default", 9090, cm, broker, nil, func(cert *certificate.Certificate) error {
		stop <- struct{}{}
		cancel()
		count++
		return nil
	})
	assert.NoError(err)

	s.Run(ctx)

	cert, err := cm.IssueCertificate("my-webhook")
	assert.NoError(err)

	broker.GetCertPubSub().Pub(events.PubSubMessage{
		Kind:   announcements.CertificateRotated,
		OldObj: cert,
	})

	assert.Eventually(func() bool {
		return count > 0
	}, time.Second*2, time.Millisecond*100)
}
