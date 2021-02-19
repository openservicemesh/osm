package injector

import (
	"github.com/onsi/ginkgo"

	"github.com/openservicemesh/osm/pkg/injector/test"
)

var _ = ginkgo.Describe("Test functions creating Envoy config and rewriting the Pod's health probes to pass through Envoy", func() {

	liveness := &healthProbe{path: "/liveness", port: 81}
	readiness := &healthProbe{path: "/readiness", port: 82}
	startup := &healthProbe{path: "/startup", port: 83}

	// Listed below are the functions we are going to test.
	// The key in the map is the name of the function -- must match what's in the value of the map.
	// The key (function name) is used to locate and load the YAML file with the expected return for this function.
	functionsToTest := map[string]func() interface{}{
		"getVirtualHosts":      func() interface{} { return getVirtualHosts("/some/path", "-cluster-name-", "/original/probe/path") },
		"getProbeCluster":      func() interface{} { return getProbeCluster("cluster-name", 12341234) },
		"getLivenessCluster":   func() interface{} { return getLivenessCluster(liveness) },
		"getReadinessCluster":  func() interface{} { return getReadinessCluster(readiness) },
		"getStartupCluster":    func() interface{} { return getStartupCluster(startup) },
		"getAccessLog":         func() interface{} { return getAccessLog() },
		"getProbeListener":     func() interface{} { return getProbeListener("a", "b", "c", 9, liveness) },
		"getLivenessListener":  func() interface{} { return getLivenessListener(liveness) },
		"getReadinessListener": func() interface{} { return getReadinessListener(readiness) },
		"getStartupListener":   func() interface{} { return getStartupListener(startup) },
	}

	for fnName, fn := range functionsToTest {
		// A call to test.ThisFunction will:
		//     a) marshal the output of each function (and save it to "actual_output_<functionName>.yaml")
		//     b) load expectation from "expected_output_<functionName>.yaml"
		//     c) compare actual and expected in a ginkgo.Context() + ginkgo.It()
		test.ThisFunction(fnName, fn)
	}
})
