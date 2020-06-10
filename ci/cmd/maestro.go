package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/open-service-mesh/osm/ci/cmd/maestro"
	"github.com/open-service-mesh/osm/demo/cmd/common"
	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/logger"
)

var log = logger.New("ci/maestro")

const (
	bookBuyerLabel = "bookbuyer"
	bookThiefLabel = "bookthief"
	selectorKey    = "app"
)

var (
	adsPodSelector    = fmt.Sprintf("%s=%s", selectorKey, constants.AggregatedDiscoveryServiceName)
	bookThiefSelector = fmt.Sprintf("%s=%s", selectorKey, bookThiefLabel)
	bookBuyerSelector = fmt.Sprintf("%s=%s", selectorKey, bookBuyerLabel)

	osmNamespace     = os.Getenv(maestro.OSMNamespaceEnvVar)
	bookbuyerNS      = os.Getenv(maestro.BookbuyerNamespaceEnvVar)
	bookthiefNS      = os.Getenv(maestro.BookthiefNamespaceEnvVar)
	bookstoreNS      = os.Getenv(maestro.BookstoreNamespaceEnvVar)
	maxPodWaitString = common.GetEnv(maestro.WaitForPodTimeSecondsEnvVar, "30")
	maxOKWaitString  = common.GetEnv(maestro.WaitForOKSecondsEnvVar, "30")
	osmID            = osmNamespace

	namespaces = []string{
		bookbuyerNS,
		bookthiefNS,
		bookstoreNS,
		osmNamespace,
	}
)

func main() {
	if bookbuyerNS == "" || bookthiefNS == "" {
		log.Error().Msgf("Namespace cannot be empty, bookbuyer=%s, bookthief=%s", bookbuyerNS, bookthiefNS)
		os.Exit(1)
	}

	log.Info().Msgf("Looking for: %s/%s and %s/%s", bookBuyerLabel, bookbuyerNS, bookThiefLabel, bookthiefNS)

	kubeClient := maestro.GetKubernetesClient()

	// Wait for pods to be ready
	{
		var wg sync.WaitGroup

		wg.Add(1)
		go maestro.WaitForPodToBeReady(kubeClient, maxWaitForPod(), bookthiefNS, bookThiefSelector, &wg)

		wg.Add(1)
		go maestro.WaitForPodToBeReady(kubeClient, maxWaitForPod(), bookbuyerNS, bookBuyerSelector, &wg)

		wg.Wait()
	}

	bookBuyerPodName, err := maestro.GetPodName(kubeClient, bookbuyerNS, bookBuyerSelector)
	if err != nil {
		fmt.Println("Error getting bookbuyer pod after pod being ready: ", err)
		os.Exit(1)
	}

	bookThiefPodName, err := maestro.GetPodName(kubeClient, bookthiefNS, bookThiefSelector)
	if err != nil {
		fmt.Println("Error getting bookthief pod after pod being ready: ", err)
		os.Exit(1)
	}

	// Tail the logs of the BookBuyer and BookThief pods concurrently and watch for success or failure.
	bookBuyerCh := make(chan maestro.TestResult)
	maestro.SearchLogsForSuccess(kubeClient, bookbuyerNS, bookBuyerPodName, bookBuyerLabel, maxWaitForOK(), bookBuyerCh)

	bookThiefCh := make(chan maestro.TestResult)
	maestro.SearchLogsForSuccess(kubeClient, bookthiefNS, bookThiefPodName, bookThiefLabel, maxWaitForOK(), bookThiefCh)

	bookBuyerTestResult := <-bookBuyerCh
	bookThiefTestResult := <-bookThiefCh

	// When both pods return success - easy - we are good to go! CI passed!
	if bookBuyerTestResult == maestro.TestsPassed && bookThiefTestResult == maestro.TestsPassed {
		log.Info().Msg("Test succeeded")
		maestro.DeleteNamespaces(kubeClient, namespaces...)
		webhookName := fmt.Sprintf("osm-webhook-%s", osmID)
		maestro.DeleteWebhook(kubeClient, webhookName)
		os.Exit(0)
	}

	// One or both of the pods did not return success.
	// Figure out what happened and print an informative message.
	humanize := map[maestro.TestResult]string{
		maestro.TestsFailed:   "failed",
		maestro.TestsTimedOut: "timedout",
	}

	if bookBuyerTestResult != maestro.TestsPassed {
		log.Error().Msgf("Bookbuyer test %s", humanize[bookBuyerTestResult])
	}

	if bookThiefTestResult != maestro.TestsPassed {
		log.Error().Msgf("BookThief test %s", humanize[bookThiefTestResult])
	}

	fmt.Println("The integration test failed")

	bookBuyerLogs := maestro.GetPodLogs(kubeClient, bookbuyerNS, bookBuyerPodName, bookBuyerLabel, maestro.FailureLogsFromTimeSince)
	bookThiefLogs := maestro.GetPodLogs(kubeClient, bookthiefNS, bookThiefPodName, bookThiefLabel, maestro.FailureLogsFromTimeSince)
	fmt.Println("-------- Bookbuyer LOGS --------\n", cutIt(bookBuyerLogs))
	fmt.Println("-------- Bookthief LOGS --------\n", cutIt(bookThiefLogs))

	osmPodName, err := maestro.GetPodName(kubeClient, osmNamespace, adsPodSelector)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error getting ADS pods with selector %s in namespace %s", osmPodName, osmNamespace)
	}

	fmt.Println("-------- ADS LOGS --------\n", maestro.GetPodLogs(kubeClient, osmNamespace, osmPodName, "", maestro.FailureLogsFromTimeSince))

	os.Exit(1)
}

func cutItAt(logs string, at string) string {
	firstOccurrence := strings.Index(logs, at)
	if firstOccurrence == -1 {
		return logs
	}
	return logs[:firstOccurrence+len(at)]
}

func cutIt(logs string) string {
	firstSuccess := strings.Index(logs, common.Success)
	if firstSuccess == -1 {
		return cutItAt(logs, common.Failure)
	}
	return cutItAt(logs, common.Success)
}

func maxWaitForPod() time.Duration {
	maxWaitInt, err := strconv.ParseInt(maxPodWaitString, 10, 32)
	if err != nil {
		log.Fatal().Err(err).Msgf("Could not convert environment variable %s='%s' to int", maestro.WaitForPodTimeSecondsEnvVar, maxPodWaitString)
	}
	return time.Duration(maxWaitInt) * time.Second
}

func maxWaitForOK() time.Duration {
	maxWaitInt, err := strconv.ParseInt(maxOKWaitString, 10, 32)
	if err != nil {
		log.Fatal().Err(err).Msgf("Could not convert environment variable %s='%s' to int", maestro.WaitForOKSecondsEnvVar, maxOKWaitString)
	}
	return time.Duration(maxWaitInt) * time.Second
}
