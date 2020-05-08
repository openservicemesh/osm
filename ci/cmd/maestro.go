package main

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/open-service-mesh/osm/ci/cmd/maestro"
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

	osmNamespace  = os.Getenv(maestro.OSMNamespaceEnvVar)
	bookbuyerNS   = os.Getenv(maestro.BookbuyerNamespaceEnvVar)
	bookthiefNS   = os.Getenv(maestro.BookthiefNamespaceEnvVar)
	bookstoreNS   = os.Getenv(maestro.BookstoreNamespaceEnvVar)
	maxWaitString = os.Getenv(maestro.WaitForPodTimeSecondsEnvVar)
	osmID         = os.Getenv(maestro.OsmIDEnvVar)

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
		go maestro.WaitForPodToBeReady(kubeClient, maxWait(), bookthiefNS, bookThiefSelector, &wg)

		wg.Add(1)
		go maestro.WaitForPodToBeReady(kubeClient, maxWait(), bookbuyerNS, bookBuyerSelector, &wg)

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
	maestro.SearchLogsForSuccess(kubeClient, bookbuyerNS, bookBuyerPodName, bookBuyerLabel, maxWait(), bookBuyerCh)

	bookThiefCh := make(chan maestro.TestResult)
	maestro.SearchLogsForSuccess(kubeClient, bookthiefNS, bookThiefPodName, bookThiefLabel, maxWait(), bookThiefCh)

	bookBuyerSucceeded := <-bookBuyerCh
	bookThiefSucceeded := <-bookThiefCh

	// When both pods return success - easy - we are good to go! CI passed!
	if bookBuyerSucceeded == maestro.TestsPassed && bookThiefSucceeded == maestro.TestsPassed {
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

	if bookBuyerSucceeded != maestro.TestsPassed {
		log.Error().Msgf("Bookbuyer test %s", humanize[bookThiefSucceeded])
	}

	if bookThiefSucceeded != maestro.TestsPassed {
		log.Error().Msgf("BookThief test %s", humanize[bookThiefSucceeded])
	}

	fmt.Println("The integration test failed")

	bookBuyerLogs := maestro.GetPodLogs(kubeClient, bookbuyerNS, bookBuyerPodName, bookBuyerLabel, maestro.FailureLogsFromTimeSince)
	bookThiefLogs := maestro.GetPodLogs(kubeClient, bookthiefNS, bookThiefPodName, bookThiefLabel, maestro.FailureLogsFromTimeSince)
	fmt.Println("-------- Bookbuyer LOGS --------\n", bookBuyerLogs)
	fmt.Println("-------- Bookthief LOGS --------\n", bookThiefLogs)

	osmPodName, err := maestro.GetPodName(kubeClient, osmNamespace, adsPodSelector)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error getting ADS pods with selector %s in namespace %s", osmPodName, osmNamespace)
	}

	fmt.Println("-------- ADS LOGS --------\n", maestro.GetPodLogs(kubeClient, osmNamespace, osmPodName, "", maestro.FailureLogsFromTimeSince))

	os.Exit(1)
}

func maxWait() time.Duration {
	maxWaitInt, err := strconv.ParseInt(maxWaitString, 10, 32)
	if err != nil {
		log.Fatal().Err(err).Msgf("Could not convert environment variable %s='%s' to int", maestro.WaitForPodTimeSecondsEnvVar, maxWaitString)
	}
	return time.Duration(maxWaitInt) * time.Second
}
