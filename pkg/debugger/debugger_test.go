package debugger

import (
	"sort"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
)

var _ = Describe("Test debugger server", func() {
	kubeClient := testclient.NewSimpleClientset()
	cache := make(map[certificate.CommonName]certificate.Certificater)
	certManager := tresor.NewFakeCertManager(&cache, 1*time.Hour)
	meshCatalog := catalog.NewFakeMeshCatalog(kubeClient)
	cfg := configurator.NewFakeConfigurator()

	debugServer := NewDebugServer(certManager, nil, meshCatalog, nil, kubeClient, cfg)

	Context("Testing debugger.GetHandlers()", func() {
		It("returns the list of handlers", func() {
			var actual []string
			for debugEndpoint := range debugServer.GetHandlers() {
				actual = append(actual, debugEndpoint)
			}
			expected := []string{
				"/debug",
				"/debug/certs",
				"/debug/config",
				"/debug/policies",
				"/debug/proxy",
				"/debug/xds",
			}
			sort.Strings(actual)
			Expect(actual).To(Equal(expected))
		})
	})
})
