package catalog

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/types"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
)

var _ = Describe("Test Announcement Handlers", func() {
	var mc *MeshCatalog
	var podUID string
	var proxy *envoy.Proxy
	var envoyCN certificate.CommonName

	BeforeEach(func() {
		mc = NewFakeMeshCatalog(testclient.NewSimpleClientset())
		podUID = uuid.New().String()

		envoyCN = "abcdefg"
		_, err := mc.certManager.IssueCertificate(envoyCN, 5*time.Second)
		Expect(err).ToNot(HaveOccurred())

		proxy = envoy.NewProxy(envoyCN, nil)
		proxy.PodMetadata = &envoy.PodMetadata{
			UID: podUID,
		}

		mc.RegisterProxy(proxy)
	})

	Context("test releaseCertificate()", func() {
		It("deletes certificate when Pod is terminated", func() {
			// Ensure setup is correct
			{
				certs, err := mc.certManager.ListCertificates()
				Expect(err).ToNot(HaveOccurred())
				Expect(len(certs)).To(Equal(1))
			}

			ann := announcements.Announcement{
				Type:               announcements.PodDeleted,
				ReferencedObjectID: types.UID(podUID),
			}
			err := mc.releaseCertificate(ann)
			Expect(err).ToNot(HaveOccurred())

			// Ensure certificate was deleted
			{
				certs, err := mc.certManager.ListCertificates()
				Expect(err).ToNot(HaveOccurred())
				Expect(len(certs)).To(Equal(0))
			}
		})

		It("ignores events other than pod-deleted", func() {
			ann := announcements.Announcement{
				Type: announcements.IngressAdded,
			}

			var connectedProxies []envoy.Proxy
			mc.connectedProxies.Range(func(key interface{}, value interface{}) bool {
				connectedProxy := value.(connectedProxy)
				connectedProxies = append(connectedProxies, *connectedProxy.proxy)
				return true
			})

			Expect(len(connectedProxies)).To(Equal(1))
			Expect(connectedProxies[0]).To(Equal(*proxy))

			err := mc.releaseCertificate(ann)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("test updateRelatedProxies()", func() {
		It("ignores events other than pod-deleted", func() {
			ann := announcements.Announcement{
				Type: announcements.IngressAdded,
			}

			var connectedProxies []envoy.Proxy
			mc.connectedProxies.Range(func(key interface{}, value interface{}) bool {
				connectedProxy := value.(connectedProxy)
				connectedProxies = append(connectedProxies, *connectedProxy.proxy)
				return true
			})

			Expect(len(connectedProxies)).To(Equal(1))
			Expect(connectedProxies[0]).To(Equal(*proxy))

			err := mc.updateRelatedProxies(ann)
			Expect(err).To(HaveOccurred())
		})
	})
})
