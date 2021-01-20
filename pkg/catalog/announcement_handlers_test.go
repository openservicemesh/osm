package catalog

import (
	"time"

	"github.com/google/uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
)

var _ = Describe("Test Announcement Handlers", func() {
	var mc *MeshCatalog
	var podUID string
	var proxy *envoy.Proxy
	var envoyCN certificate.CommonName

	BeforeEach(func() {
		mc = NewFakeMeshCatalog(testclient.NewSimpleClientset())
		podUID = uuid.New().String()
		_, err := mc.certManager.IssueCertificate(envoyCN, 5*time.Second)
		Expect(err).ToNot(HaveOccurred())

		proxy = envoy.NewProxy(envoyCN, "-cert-serial-number-", nil)
		proxy.PodMetadata = &envoy.PodMetadata{
			UID: podUID,
		}

		mc.RegisterProxy(proxy)
	})

	Context("test releaseCertificate()", func() {
		var stopChannel chan struct{}
		BeforeEach(func() {
			stopChannel = mc.releaseCertificateHandler()
		})

		AfterEach(func() {
			stopChannel <- struct{}{}
		})

		It("deletes certificate when Pod is terminated", func() {
			// Ensure setup is correct
			{
				certs, err := mc.certManager.ListCertificates()
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
				certs, err := mc.certManager.ListCertificates()
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
			mc.connectedProxies.Range(func(key interface{}, value interface{}) bool {
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
			certs, err := mc.certManager.ListCertificates()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(certs)).To(Equal(1))
		})
	})
})
