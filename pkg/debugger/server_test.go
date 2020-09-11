package debugger

import (
	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/configurator"
)

var _ = Describe("Test server methods", func() {
	var (
		mockCtrl            *gomock.Controller
		mockCatalogDebugger *MockMeshCatalogDebugger
		mockCertDebugger    *MockCertificateManagerDebugger
		mockXdsDebugger     *MockXDSDebugger
		mockConfig          *configurator.MockConfigurator
		client              *testclient.Clientset
	)

	mockCtrl = gomock.NewController(GinkgoT())

	BeforeEach(func() {
		mockCatalogDebugger = NewMockMeshCatalogDebugger(mockCtrl)
		mockConfig = configurator.NewMockConfigurator(mockCtrl)
		mockCertDebugger = NewMockCertificateManagerDebugger(mockCtrl)
		client = testclient.NewSimpleClientset()
	})

	It("GetHandlers properly return the handlers of the different debug modules", func() {
		ds := NewDebugServer(mockCertDebugger,
			mockXdsDebugger,
			mockCatalogDebugger,
			nil,
			client,
			mockConfig)

		handlers := ds.GetHandlers()

		debugEndpoints := []string{
			"/debug/certs",
			"/debug/xds",
			"/debug/proxy",
			"/debug/policies",
			"/debug/config",
			"/debug/namespaces",
			// Pprof handlers
			"/debug/pprof/",
			"/debug/pprof/cmdline",
			"/debug/pprof/profile",
			"/debug/pprof/symbol",
			"/debug/pprof/trace",
		}

		for _, endpoint := range debugEndpoints {
			handler, found := handlers[endpoint]
			Expect(found).To(BeTrue())
			Expect(handler).ToNot(BeNil())
		}
	})

})
