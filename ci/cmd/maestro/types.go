package maestro

import (
	"errors"
	"time"

	"github.com/openservicemesh/osm/pkg/logger"
)

const (
	// TestsPassed is used for tests that passed.
	TestsPassed = "passed"

	// TestsFailed is used for tests that failed.
	TestsFailed = "failed"

	// TestsTimedOut is used for tests that timed out.
	TestsTimedOut = "timedout"

	// OSMNamespaceEnvVar is the environment variable for the OSM namespace.
	OSMNamespaceEnvVar = "K8S_NAMESPACE"

	// BookbuyerNamespaceEnvVar is the environment variable for the Bookbuyer namespace.
	BookbuyerNamespaceEnvVar = "BOOKBUYER_NAMESPACE"

	// BookthiefNamespaceEnvVar is the environment variable for the Bookbuyer namespace.
	BookthiefNamespaceEnvVar = "BOOKTHIEF_NAMESPACE"

	// BookstoreNamespaceEnvVar is the environment variable for the Bookbuyer namespace.
	BookstoreNamespaceEnvVar = "BOOKSTORE_NAMESPACE"

	// BookWarehouseNamespaceEnvVar is the environment variable for the BookWarehouse namespace.
	BookWarehouseNamespaceEnvVar = "BOOKWAREHOUSE_NAMESPACE"

	// WaitForPodTimeSecondsEnvVar is the environment variable for the time we will wait on the pod to be ready.
	WaitForPodTimeSecondsEnvVar = "CI_MAX_WAIT_FOR_POD_TIME_SECONDS"

	// WaitForOKSecondsEnvVar is the environment variable for the time to wait till a success is returned by the server.
	WaitForOKSecondsEnvVar = "CI_WAIT_FOR_OK_SECONDS"
)

var (
	// WaitForPod is the time we wait for a pod to become ready
	WaitForPod = 5 * time.Second

	// PollLogsFromTimeSince is the interval we go back in time to get pod logs
	PollLogsFromTimeSince = 2 * time.Second

	// FailureLogsFromTimeSince is the interval we go back in time to get pod logs
	FailureLogsFromTimeSince = 10 * time.Minute

	log            = logger.NewPretty("ci/maestro")
	errNoPodsFound = errors.New("no pods found")
)
