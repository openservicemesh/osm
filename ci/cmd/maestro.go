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

	maxPodWaitString = utils.GetEnv(maestro.WaitForPodTimeSecondsEnvVar, "30")
	maxOKWaitString  = utils.GetEnv(maestro.WaitForOKSecondsEnvVar, "30")

	// Mesh namespaces
	namespaces = []string{
		bookbuyerNS,
		bookthiefNS,
		bookstoreNS,
		bookWarehouseNS,
	}
)

func main() {
	log.Debug().Msgf("Looking for: %s/%s, %s/%s, %s/%s, %s/%s, %s/%s", bookBuyerLabel, bookbuyerNS, bookThiefLabel, bookthiefNS, bookstoreV1Label, bookstoreNS, bookstoreV2Label, bookstoreNS, bookWarehouseLabel, bookWarehouseNS)

	kubeClient := maestro.GetKubernetesClient()

	bookBuyerPodName, bookThiefPodName, bookWarehousePodName, osmControllerPodName := getPodNames(kubeClient)

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

func getPodNames(kubeClient kubernetes.Interface) (string, string, string, string) {
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
