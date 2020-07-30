package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/openservicemesh/osm/ci/cmd/maestro"
	"github.com/openservicemesh/osm/demo/cmd/common"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/logger"
)

var log = logger.NewPretty("ci/maestro")

const (
	bookBuyerLabel      = "bookbuyer"
	bookThiefLabel      = "bookthief"
	bookstoreLabel      = "bookstore"
	bookstoreV1Label    = "v1"
	bookstoreV2Label    = "v2"
	bookWarehouseLabel  = "bookwarehouse"
	selectorKey         = "app"
	bookstoreVersionKey = "version"
)

var (
	osmControllerPodSelector = fmt.Sprintf("%s=%s", selectorKey, constants.OSMControllerName)
	bookThiefSelector        = fmt.Sprintf("%s=%s", selectorKey, bookThiefLabel)
	bookBuyerSelector        = fmt.Sprintf("%s=%s", selectorKey, bookBuyerLabel)
	bookstoreV1Selector      = fmt.Sprintf("%s=%s,%s=%s", selectorKey, bookstoreLabel, bookstoreVersionKey, bookstoreV1Label)
	bookstoreV2Selector      = fmt.Sprintf("%s=%s,%s=%s", selectorKey, bookstoreLabel, bookstoreVersionKey, bookstoreV2Label)
	bookWarehouseSelector    = fmt.Sprintf("%s=%s", selectorKey, bookWarehouseLabel)

	osmNamespace    = common.GetEnv(maestro.OSMNamespaceEnvVar, "osm-system")
	bookbuyerNS     = common.GetEnv(maestro.BookbuyerNamespaceEnvVar, "bookbuyer")
	bookthiefNS     = common.GetEnv(maestro.BookthiefNamespaceEnvVar, "bookthief")
	bookstoreNS     = common.GetEnv(maestro.BookstoreNamespaceEnvVar, "bookstore")
	bookWarehouseNS = common.GetEnv(common.BookwarehouseNamespaceEnvVar, "bookwarehouse")

	maxPodWaitString = common.GetEnv(maestro.WaitForPodTimeSecondsEnvVar, "30")
	maxOKWaitString  = common.GetEnv(maestro.WaitForOKSecondsEnvVar, "30")
	meshName         = osmNamespace

	namespaces = []string{
		bookbuyerNS,
		bookthiefNS,
		bookstoreNS,
		bookWarehouseNS,
		osmNamespace,
	}
)

func main() {
	log.Info().Msgf("Looking for: %s/%s, %s/%s, %s/%s, %s/%s, %s/%s", bookBuyerLabel, bookbuyerNS, bookThiefLabel, bookthiefNS, bookstoreV1Label, bookstoreNS, bookstoreV2Label, bookstoreNS, bookWarehouseLabel, bookWarehouseNS)

	kubeClient := maestro.GetKubernetesClient()

	// Wait for pods to be ready
	{
		var wg sync.WaitGroup

		wg.Add(1)
		go maestro.WaitForPodToBeReady(kubeClient, maxWaitForPod(), bookthiefNS, bookThiefSelector, &wg)

		wg.Add(1)
		go maestro.WaitForPodToBeReady(kubeClient, maxWaitForPod(), bookbuyerNS, bookBuyerSelector, &wg)

		wg.Add(1)
		go maestro.WaitForPodToBeReady(kubeClient, maxWaitForPod(), bookstoreNS, bookstoreV1Selector, &wg)

		wg.Add(1)
		go maestro.WaitForPodToBeReady(kubeClient, maxWaitForPod(), bookstoreNS, bookstoreV2Selector, &wg)

		wg.Add(1)
		go maestro.WaitForPodToBeReady(kubeClient, maxWaitForPod(), bookWarehouseNS, bookWarehouseSelector, &wg)

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

	bookWarehousePodName, err := maestro.GetPodName(kubeClient, bookWarehouseNS, bookWarehouseSelector)
	if err != nil {
		fmt.Println("Error getting bookWarehouse pod after pod being ready: ", err)
		os.Exit(1)
	}

	// Tail the logs of the BookBuyer and BookThief pods concurrently and watch for success or failure.
	bookBuyerCh := make(chan maestro.TestResult)
	bookThiefCh := make(chan maestro.TestResult)

	maestro.SearchLogsForSuccess(kubeClient, bookbuyerNS, bookBuyerPodName, bookBuyerLabel, maxWaitForOK(), bookBuyerCh, common.Success, common.Failure)
	maestro.SearchLogsForSuccess(kubeClient, bookthiefNS, bookThiefPodName, bookThiefLabel, maxWaitForOK(), bookThiefCh, common.Success, common.Failure)

	bookWarehouseCh := make(chan maestro.TestResult)
	successToken := "Restocking bookstore with 1 new books; Total so far: 3 "
	maestro.SearchLogsForSuccess(kubeClient, bookWarehouseNS, bookWarehousePodName, bookWarehouseLabel, maxWaitForOK(), bookWarehouseCh, successToken, common.Failure)

	bookBuyerTestResult := <-bookBuyerCh
	bookThiefTestResult := <-bookThiefCh
	bookWarehouseTestResult := <-bookWarehouseCh

	// When both pods return success - easy - we are good to go! CI passed!
	if bookBuyerTestResult == maestro.TestsPassed && bookThiefTestResult == maestro.TestsPassed && bookWarehouseTestResult == maestro.TestsPassed {
		log.Info().Msg("Test succeeded")
		maestro.DeleteNamespaces(kubeClient, namespaces...)
		webhookName := fmt.Sprintf("osm-webhook-%s", meshName)
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

	if bookWarehouseTestResult != maestro.TestsPassed {
		log.Error().Msgf("BookWarehouse test %s", humanize[bookWarehouseTestResult])
	}

	fmt.Println("The integration test failed")

	bookBuyerLogs := maestro.GetPodLogs(kubeClient, bookbuyerNS, bookBuyerPodName, bookBuyerLabel, maestro.FailureLogsFromTimeSince)
	fmt.Println("-------- Bookbuyer LOGS --------\n", cutIt(bookBuyerLogs))

	bookThiefLogs := maestro.GetPodLogs(kubeClient, bookthiefNS, bookThiefPodName, bookThiefLabel, maestro.FailureLogsFromTimeSince)
	fmt.Println("-------- Bookthief LOGS --------\n", cutIt(bookThiefLogs))

	bookWarehouseLogs := maestro.GetPodLogs(kubeClient, bookWarehouseNS, bookWarehousePodName, bookWarehouseLabel, maestro.FailureLogsFromTimeSince)
	fmt.Println("-------- BookWarehouse LOGS --------\n", cutIt(bookWarehouseLogs))

	osmPodName, err := maestro.GetPodName(kubeClient, osmNamespace, osmControllerPodSelector)

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
