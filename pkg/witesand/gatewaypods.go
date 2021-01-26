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

func (wc *WitesandCatalog) ListLocalGatewayPods() (*ClusterPods, error) {
	kubeClient := wc.kubeClient
	svcName := "gateway"

	podList, err := kubeClient.CoreV1().Pods("default").List(context.Background(), v12.ListOptions{})
	if err != nil {
		log.Error().Err(err).Msgf("Error listing pods in namespace %s", "default")
		return nil, fmt.Errorf("error listing pod")
	}

	pods := ClusterPods{
			PodToIPMap: make(map[string]string),
		}
	for _, pod := range podList.Items {
		if strings.HasPrefix(pod.Name, svcName) && pod.Status.Phase == "Running" {
			log.Info().Msgf("pod.Name=%+v, pod.PodIP=%+v \n", pod.Name, pod.Status.PodIP)
			pods.PodToIPMap[pod.Name] = pod.Status.PodIP
		}
	}
	return &pods, nil
}

func (wc *WitesandCatalog) ListAllGatewayPods() ([]string, error) {
	pods := make([]string, 0)
	for _, clusterPods := range wc.clusterPodMap {
		for pod, _ := range clusterPods.PodToIPMap {
			pods = append(pods, pod)
		}
	}

	return pods, nil
}

func (wc *WitesandCatalog) ListAllGatewayPodIPs() (*ClusterPods, error) {
	pods := ClusterPods{
			PodToIPMap: make(map[string]string),
		}
	for _, clusterPods := range wc.clusterPodMap {
		for podName, podIP := range clusterPods.PodToIPMap {
			pods.PodToIPMap[podName] = podIP
		}
	}

	return &pods, nil
}

func (wc *WitesandCatalog) UpdateClusterPods(clusterId string, clusterPods *ClusterPods) {
	log.Info().Msgf("[UpdateClusterPods] clusterId:%s clusterPods:%+v", clusterId, *clusterPods)
	// checks to see if anything (pod or podip) has changed to trigger update
	triggerUpdate := false
	prevClusterPods, exists := wc.clusterPodMap[clusterId]
	if exists && clusterPods != nil && len(prevClusterPods.PodToIPMap) == len(clusterPods.PodToIPMap) {
		for pod, podip := range clusterPods.PodToIPMap {
			prevPodIp, exists := prevClusterPods.PodToIPMap[pod]
			if !exists || prevPodIp != podip {
				triggerUpdate = true
				break
			}
		}
	} else {
		triggerUpdate = true
	}

	// LOCK
	if triggerUpdate {
		log.Info().Msgf("[UpdateClusterPods] triggering update")
		if clusterPods == nil || len(clusterPods.PodToIPMap) == 0 {
			delete(wc.clusterPodMap, clusterId)
		} else {
			wc.clusterPodMap[clusterId] = *clusterPods
		}
		wc.updateEnvoy()
	}
}
