package injector

import (
	"testing"
	"time"

	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/openservicemesh/osm/pkg/injector/test"
)

var _ = ginkgo.Describe("Test functions creating Envoy config and rewriting the Pod's health probes to pass through Envoy", func() {

	timeout := 42 * time.Second
	liveness := &healthProbe{path: "/liveness", port: 81, isHTTP: true, isTCPSocket: false, timeout: timeout}
	livenessNonHTTP := &healthProbe{port: 81, isHTTP: false, isTCPSocket: false, timeout: timeout}
	readiness := &healthProbe{path: "/readiness", port: 82, isHTTP: true, isTCPSocket: false, timeout: timeout}
	startup := &healthProbe{path: "/startup", port: 83, isHTTP: true, isTCPSocket: false, timeout: timeout}

	// Listed below are the functions we are going to test.
	// The key in the map is the name of the function -- must match what's in the value of the map.
	// The key (function name) is used to locate and load the YAML file with the expected return for this function.
	clusterFunctionsToTest := map[string]func() protoreflect.ProtoMessage{
		"getVirtualHosts": func() protoreflect.ProtoMessage {
			return getVirtualHost("/some/path", "-cluster-name-", "/original/probe/path", timeout)
		},
		"getVirtualHostsDefault": func() protoreflect.ProtoMessage {
			return getVirtualHost("/some/path", "-cluster-name-", "/original/probe/path", 0*time.Second)
		},
		"getProbeCluster":     func() protoreflect.ProtoMessage { return getProbeCluster("cluster-name", 12341234) },
		"getLivenessCluster":  func() protoreflect.ProtoMessage { return getLivenessCluster(liveness) },
		"getReadinessCluster": func() protoreflect.ProtoMessage { return getReadinessCluster(readiness) },
		"getStartupCluster":   func() protoreflect.ProtoMessage { return getStartupCluster(startup) },
	}

	listenerFunctionsToTest := map[string]func() (protoreflect.ProtoMessage, error){
		"getHTTPAccessLog":           func() (protoreflect.ProtoMessage, error) { return getHTTPAccessLog() },
		"getTCPAccessLog":            func() (protoreflect.ProtoMessage, error) { return getTCPAccessLog() },
		"getProbeListener":           func() (protoreflect.ProtoMessage, error) { return getProbeListener("a", "b", "c", 9, liveness) },
		"getLivenessListener":        func() (protoreflect.ProtoMessage, error) { return getLivenessListener(liveness) },
		"getLivenessListenerNonHTTP": func() (protoreflect.ProtoMessage, error) { return getLivenessListener(livenessNonHTTP) },
		"getReadinessListener":       func() (protoreflect.ProtoMessage, error) { return getReadinessListener(readiness) },
		"getStartupListener":         func() (protoreflect.ProtoMessage, error) { return getStartupListener(startup) },
	}

	for fnName, fn := range clusterFunctionsToTest {
		// A call to test.ThisFunction will:
		//     a) marshal return xDS struct of each function to yaml (and save it to "actual_output_<functionName>.yaml")
		//     b) load expectation from "expected_output_<functionName>.yaml"
		//     c) compare actual and expected in a ginkgo.Context() + ginkgo.It()
		test.ThisXdsClusterFunction(fnName, fn)
	}

	for fnName, fn := range listenerFunctionsToTest {
		// A call to test.ThisFunction will:
		//     a) check for error
		//     b) marshal return xDS struct of each function to yaml (and save it to "actual_output_<functionName>.yaml")
		//     c) load expectation from "expected_output_<functionName>.yaml"
		//     d) compare actual and expected in a ginkgo.Context() + ginkgo.It()
		test.ThisXdsListenerFunction(fnName, fn)
	}
})

func TestGetProbeCluster(t *testing.T) {
	type probeClusterTest struct {
		name     string
		probe    *healthProbe
		expected *xds_cluster.Cluster
	}

	t.Run("liveness", func(t *testing.T) {
		tests := []probeClusterTest{
			{
				name: "nil",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				assert.Equal(t, test.expected, getLivenessCluster(test.probe))
			})
		}
	})

	t.Run("readiness", func(t *testing.T) {
		tests := []probeClusterTest{
			{
				name: "nil",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				assert.Equal(t, test.expected, getReadinessCluster(test.probe))
			})
		}
	})

	t.Run("startup", func(t *testing.T) {
		tests := []probeClusterTest{
			{
				name: "nil",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				assert.Equal(t, test.expected, getStartupCluster(test.probe))
			})
		}
	})
}

func TestGetProbeListener(t *testing.T) {
	type probeListenerTest struct {
		name     string
		probe    *healthProbe
		expected *xds_listener.Listener
		err      error
	}

	t.Run("liveness", func(t *testing.T) {
		tests := []probeListenerTest{
			{
				name: "nil",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				actual, err := getLivenessListener(test.probe)
				assert.Equal(t, test.expected, actual)
				assert.Equal(t, test.err, err)
			})
		}
	})

	t.Run("readiness", func(t *testing.T) {
		tests := []probeListenerTest{
			{
				name: "nil",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				actual, err := getReadinessListener(test.probe)
				assert.Equal(t, test.expected, actual)
				assert.Equal(t, test.err, err)
			})
		}
	})

	t.Run("startup", func(t *testing.T) {
		tests := []probeListenerTest{
			{
				name: "nil",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				actual, err := getStartupListener(test.probe)
				assert.Equal(t, test.expected, actual)
				assert.Equal(t, test.err, err)
			})
		}
	})
}
