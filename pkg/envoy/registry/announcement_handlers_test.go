package registry

import (
	"time"

	"github.com/google/uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
)

var _ = Describe("Test Announcement Handlers", func() {
	var proxyRegistry *ProxyRegistry
	var podUID string
	var proxy *envoy.Proxy
	var envoyCN certificate.CommonName
	var certManager certificate.Manager

	BeforeEach(func() {
		proxyRegistry = NewProxyRegistry()
		podUID = uuid.New().String()

		stop := make(<-chan struct{})
		kubeClient := fake.NewSimpleClientset()
		osmNamespace := "-test-osm-namespace-"
		osmConfigMapName := "-test-osm-config-map-"
		cfg := configurator.NewConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)
		certManager = tresor.NewFakeCertManager(cfg)

		_, err := certManager.IssueCertificate(envoyCN, 5*time.Second)
		Expect(err).ToNot(HaveOccurred())

		proxy = envoy.NewProxy(envoyCN, "-cert-serial-number-", nil)
		proxy.PodMetadata = &envoy.PodMetadata{
			UID: podUID,
		}

		proxyRegistry.RegisterProxy(proxy)
	})

	Context("test releaseCertificate()", func() {
		var stopChannel chan struct{}
		BeforeEach(func() {
			stopChannel = proxyRegistry.ReleaseCertificateHandler(certManager)
		})

		AfterEach(func() {
			stopChannel <- struct{}{}
		})

		It("deletes certificate when Pod is terminated", func() {
			// Ensure setup is correct
			{
				certs, err := certManager.ListCertificates()
				Expect(err).ToNot(HaveOccurred())
				Expect(len(certs)).To(Equal(1))
			}

			// Register to Update proxies event. We should see a schedule broadcast update
			// requested by the handler when the certificate is released.
			rcvBroadcastChannel := events.GetPubSubInstance().Subscribe(announcements.ScheduleProxyBroadcast)

			// Publish a podDeleted event
			events.GetPubSubInstance().Publish(events.PubSubMessage{
				AnnouncementType: announcements.PodDeleted,
				NewObj:           nil,
				OldObj: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						UID: types.UID(podUID),
					},
				},
			})

			// Expect the certificate to eventually be gone for the deleted Pod
			Eventually(func() int {
				certs, err := certManager.ListCertificates()
				Expect(err).ToNot(HaveOccurred())
				return len(certs)
			}).Should(Equal(0))

			select {
			case <-rcvBroadcastChannel:
				// broadcast event received
			case <-time.After(1 * time.Second):
				Fail("Did not see a broadcast request in time")
			}
		})

		It("ignores events other than pod-deleted", func() {
			var connectedProxies []envoy.Proxy
			proxyRegistry.connectedProxies.Range(func(key interface{}, value interface{}) bool {
				connectedProxy := value.(connectedProxy)
				connectedProxies = append(connectedProxies, *connectedProxy.proxy)
				return true // continue the iteration
			})

			Expect(len(connectedProxies)).To(Equal(1))
			Expect(connectedProxies[0]).To(Equal(*proxy))

			// Publish some event unrelated to podDeleted
			events.GetPubSubInstance().Publish(events.PubSubMessage{
				AnnouncementType: announcements.IngressAdded,
				NewObj:           nil,
				OldObj: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						UID: types.UID(proxy.PodMetadata.UID),
					},
				},
			})

			// Give some grace period for event to propagate
			time.Sleep(500 * time.Millisecond)

			// Ensure it was not deleted due to an unrelated event
			certs, err := certManager.ListCertificates()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(certs)).To(Equal(1))
		})
	})
})
