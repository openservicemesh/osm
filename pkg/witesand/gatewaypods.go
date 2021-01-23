package witesand

import (
	"context"
	"fmt"
	"strings"

	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (wc *WitesandCatalog) GetClusterId() string {
	return wc.clusterId
}

func (wc *WitesandCatalog) ListLocalGatewaypods() ([]string, error) {
	kubeClient := wc.kubeClient
	svcName := GatewayServiceName

	podList, err := kubeClient.CoreV1().Pods("default").List(context.Background(), v12.ListOptions{})
	if err != nil {
		log.Error().Err(err).Msgf("Error listing pods in namespace %s", "default")
		return nil, fmt.Errorf("error listing pod")
	}

	searchList := make([]string, 0)
	for _, pod := range podList.Items {
		if strings.Contains(pod.Name, svcName) && pod.Status.Phase == "Running" {
			log.Info().Msgf("pod.Name=%+v, pod.status=%+v \n", pod.Name, pod.Status.Phase)
			searchList = append(searchList, pod.Name)
		}
	}
	return searchList, nil
}

func (wc *WitesandCatalog) ListAllGatewaypods() ([]string, error) {
	pods, err := wc.ListLocalGatewaypods()
	if err != nil {
		return pods, err
	}

	for _, remotePods := range wc.remotePodMap {
		for pod, _ := range remotePods.PodToIPMap {
			pods = append(pods, pod)
		}
	}

	return pods, nil
}

func (wc *WitesandCatalog) UpdateRemotePods(clusterId string, remotePods *RemotePods) {
	// LOCK
	wc.remotePodMap[clusterId] = *remotePods
}
