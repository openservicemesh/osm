package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/ci/cmd/maestro"
	"github.com/openservicemesh/osm/demo/cmd/common"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/logger"
	osmStrings "github.com/openservicemesh/osm/pkg/strings"
	"github.com/openservicemesh/osm/pkg/utils"
)

var log = logger.NewPretty("ci/maestro")

const (
	// Pod labels
	bookBuyerLabel     = "bookbuyer"
	bookThiefLabel     = "bookthief"
	bookstoreV1Label   = "bookstore-v1"
	bookstoreV2Label   = "bookstore-v2"
	bookWarehouseLabel = "bookwarehouse"

	selectorKey = "app"
)

var (
	osmControllerPodSelector = fmt.Sprintf("%s=%s", selectorKey, constants.OSMControllerName)
	bookThiefSelector        = fmt.Sprintf("%s=%s", selectorKey, bookThiefLabel)
	bookBuyerSelector        = fmt.Sprintf("%s=%s", selectorKey, bookBuyerLabel)
	bookstoreV1Selector      = fmt.Sprintf("%s=%s", selectorKey, bookstoreV1Label)
	bookstoreV2Selector      = fmt.Sprintf("%s=%s", selectorKey, bookstoreV2Label)
	bookWarehouseSelector    = fmt.Sprintf("%s=%s", selectorKey, bookWarehouseLabel)

	osmNamespace    = utils.GetEnv(maestro.OSMNamespaceEnvVar, "osm-system")
	bookbuyerNS     = utils.GetEnv(maestro.BookbuyerNamespaceEnvVar, "bookbuyer")
	bookthiefNS     = utils.GetEnv(maestro.BookthiefNamespaceEnvVar, "bookthief")
	bookstoreNS     = utils.GetEnv(maestro.BookstoreNamespaceEnvVar, "bookstore")
	bookWarehouseNS = utils.GetEnv(common.BookwarehouseNamespaceEnvVar, "bookwarehouse")

	maxPodWaitString        = utils.GetEnv(maestro.WaitForPodTimeSecondsEnvVar, "30")
	maxOKWaitString         = utils.GetEnv(maestro.WaitForOKSecondsEnvVar, "30")
	multiClusterModeEnabled = utils.GetEnv(maestro.MulticlusterModeEnvVar, "false")

	// Mesh namespaces
	namespaces = []string{
		bookbuyerNS,
		bookthiefNS,
		bookstoreNS,
		bookWarehouseNS,
	}
)

func main() {
	log.Debug().Msgf("Multicluster mode: %s", multiClusterModeEnabled)

	if multiClusterModeEnabled == "true" {
		testMultiCluster()
	} else {
		testSingleCluster()
	}

	os.Exit(1)
}

func testSingleCluster() {
	log.Debug().Msgf("Looking for: %s/%s, %s/%s, %s/%s, %s/%s, %s/%s", bookBuyerLabel, bookbuyerNS, bookThiefLabel, bookthiefNS, bookstoreV1Label, bookstoreNS, bookstoreV2Label, bookstoreNS, bookWarehouseLabel, bookWarehouseNS)

	kubeClient := maestro.GetKubernetesClient()

	bookBuyerPodName, bookThiefPodName, bookWarehousePodName, osmControllerPodName := getPodNames(kubeClient, true, true, true, true, true)

	// Tail the logs of the pods participating in the service mesh concurrently and watch for success or failure.
	didItSucceed := func(ns, podName, label string) chan string {
		result := make(chan string)
		maestro.SearchLogsForSuccess(kubeClient, ns, podName, label, maxWaitForOK(), result, common.Success, common.Failure)
		return result
	}

	// When both pods return success - easy - we are good to go! CI passed!
	allTestsResults := osmStrings.All{
		<-didItSucceed(bookbuyerNS, bookBuyerPodName, bookBuyerLabel),
		<-didItSucceed(bookthiefNS, bookThiefPodName, bookThiefLabel),
		<-didItSucceed(bookWarehouseNS, bookWarehousePodName, bookWarehouseLabel),
	}

	if allTestsResults.Equal(maestro.TestsPassed) {
		log.Debug().Msg("Test succeeded")
		maestro.DeleteNamespaces(kubeClient, append(namespaces, osmNamespace)...)
		os.Exit(0) // Tests passed!  WE ARE DONE !!!
	}

	if failedTests := osmStrings.Which(allTestsResults).NotEqual(maestro.TestsPassed); len(failedTests) != 0 {
		log.Error().Msgf("%s did not pass; Retrieving pod logs", strings.Join(failedTests, ","))
	}

	// Walk mesh-participant namespaces
	for _, ns := range namespaces {
		pods, err := kubeClient.CoreV1().Pods(ns).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			log.Error().Err(err).Msgf("Could not get Pods for Namespace %s", ns)
			continue
		}

		for _, pod := range pods.Items {
			printLogsForInitContainers(kubeClient, pod)
			printLogsForContainers(kubeClient, pod)
		}
	}
	fmt.Println("-------- OSM-Controller LOGS --------\n",
		maestro.GetPodLogs(kubeClient, osmNamespace, osmControllerPodName, "", maestro.FailureLogsFromTimeSince))
}

func testMultiCluster() {
	log.Debug().Msgf("Looking for: %s/%s, %s/%s", bookBuyerLabel, bookbuyerNS, bookstoreV1Label, bookstoreNS)

	kubeClient := maestro.GetKubernetesClient()

	bookBuyerPodName, _, _, osmControllerPodName := getPodNames(kubeClient, false, true, true, false, false)

	// Tail the logs of the pods participating in the service mesh concurrently and watch for success or failure..
	didItSucceed := func(ns, podName, label string) (chan string, chan string) {
		resultAlpha := make(chan string)
		resultBeta := make(chan string)

		// multicluster mode
		alphaClusterName := utils.GetEnv(maestro.AlphaClusterEnvVar, "alpha")
		betaClusterName := utils.GetEnv(maestro.BetaClusterEnvVar, "beta")

		maestro.SearchLogsForSuccess(kubeClient, ns, podName, label, maxWaitForOK(), resultAlpha, fmt.Sprintf("Identity: bookstore-v1.%s", alphaClusterName), common.NoToken)
		maestro.SearchLogsForSuccess(kubeClient, ns, podName, label, maxWaitForOK(), resultBeta, fmt.Sprintf("Identity: bookstore-v1.%s", betaClusterName), common.NoToken)

		return resultAlpha, resultBeta
	}

	resultAlpha, resultBeta := didItSucceed(bookbuyerNS, bookBuyerPodName, bookBuyerLabel)
	testResultAlpha := <-resultAlpha
	testResultBeta := <-resultBeta

	if testResultAlpha == maestro.TestsPassed && testResultBeta == maestro.TestsPassed {
		log.Debug().Msg("Test succeeded")
		maestro.DeleteNamespaces(kubeClient, append(namespaces, osmNamespace)...)
		os.Exit(0) // Tests passed!  WE ARE DONE !!!
	}

	log.Error().Msgf("Test did not pass; Retrieving bookbuyer logs")

	// Walk mesh-participant namespaces
	for _, ns := range namespaces {
		pods, err := kubeClient.CoreV1().Pods(ns).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			log.Error().Err(err).Msgf("Could not get Pods for Namespace %s", ns)
			continue
		}

		for _, pod := range pods.Items {
			printLogsForInitContainers(kubeClient, pod)
			printLogsForContainers(kubeClient, pod)
		}
	}
	fmt.Println("-------- OSM-Controller LOGS --------\n",
		maestro.GetPodLogs(kubeClient, osmNamespace, osmControllerPodName, "", maestro.FailureLogsFromTimeSince))

	maestro.DeleteNamespaces(kubeClient, append(namespaces, osmNamespace)...)
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

func getPodNames(kubeClient kubernetes.Interface, includeBookthief, includeBookbuyer, includeBookstoreV1, includeBookstoreV2, includeBookwarehouse bool) (string, string, string, string) {
	var wg sync.WaitGroup

	if includeBookthief {
		wg.Add(1)
		go maestro.WaitForPodToBeReady(kubeClient, maxWaitForPod(), bookthiefNS, bookThiefSelector, &wg)
	}

	if includeBookbuyer {
		wg.Add(1)
		go maestro.WaitForPodToBeReady(kubeClient, maxWaitForPod(), bookbuyerNS, bookBuyerSelector, &wg)
	}

	if includeBookstoreV1 {
		wg.Add(1)
		go maestro.WaitForPodToBeReady(kubeClient, maxWaitForPod(), bookstoreNS, bookstoreV1Selector, &wg)
	}

	if includeBookstoreV2 {
		wg.Add(1)
		go maestro.WaitForPodToBeReady(kubeClient, maxWaitForPod(), bookstoreNS, bookstoreV2Selector, &wg)
	}

	if includeBookwarehouse {
		wg.Add(1)
		go maestro.WaitForPodToBeReady(kubeClient, maxWaitForPod(), bookWarehouseNS, bookWarehouseSelector, &wg)
	}

	wg.Wait()

	var bookBuyerPodName, bookThiefPodName, bookWarehousePodName string
	var err error
	if includeBookbuyer {
		bookBuyerPodName, err = maestro.GetPodName(kubeClient, bookbuyerNS, bookBuyerSelector)
		if err != nil {
			fmt.Println("Error getting bookbuyer pod after pod being ready: ", err)
			os.Exit(1)
		}
	}

	if includeBookthief {
		bookThiefPodName, err = maestro.GetPodName(kubeClient, bookthiefNS, bookThiefSelector)
		if err != nil {
			fmt.Println("Error getting bookthief pod after pod being ready: ", err)
			os.Exit(1)
		}
	}

	if includeBookwarehouse {
		bookWarehousePodName, err = maestro.GetPodName(kubeClient, bookWarehouseNS, bookWarehouseSelector)
		if err != nil {
			fmt.Println("Error getting bookWarehouse pod after pod being ready: ", err)
			os.Exit(1)
		}
	}

	osmControllerPodName, err := maestro.GetPodName(kubeClient, osmNamespace, osmControllerPodSelector)
	if err != nil {
		fmt.Println("Error getting bookWarehouse pod after pod being ready: ", err)
		os.Exit(1)
	}

	return bookBuyerPodName, bookThiefPodName, bookWarehousePodName, osmControllerPodName
}

func printLogsForInitContainers(kubeClient kubernetes.Interface, pod v1.Pod) {
	for _, initContainer := range pod.Spec.InitContainers {
		initLogs := maestro.GetPodLogs(kubeClient, pod.Namespace, pod.Name, initContainer.Name, maestro.FailureLogsFromTimeSince)
		fmt.Println(fmt.Sprintf("---- NS: %s  Pod: %s  InitContainer: %s --------\n",
			pod.Namespace, pod.Name, initContainer.Name), cutIt(initLogs))
	}
}

func printLogsForContainers(kubeClient kubernetes.Interface, pod v1.Pod) {
	for _, containerObj := range pod.Spec.Containers {
		initLogs := maestro.GetPodLogs(kubeClient, pod.Namespace, pod.Name, containerObj.Name, maestro.FailureLogsFromTimeSince)
		switch containerObj.Name {
		case constants.EnvoyContainerName:
			fmt.Println(fmt.Sprintf("---- NS: %s  Pod: %s  Envoy Logs: --------\n",
				pod.Namespace, pod.Name), initLogs)
		default:
			fmt.Println(fmt.Sprintf("---- NS: %s  Pod: %s  Container: %s --------\n",
				pod.Namespace, pod.Name, containerObj.Name), cutIt(initLogs))
		}
	}
}
