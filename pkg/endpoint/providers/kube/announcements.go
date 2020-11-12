package kube

import (
	v1 "k8s.io/api/core/v1"
)

func getPodUID(obj interface{}) interface{} {
	if pod, ok := obj.(*v1.Pod); ok {
		return pod.ObjectMeta.UID
	}
	log.Error().Msgf("Expected a Kubernetes Pod object; Got %+v", obj)
	return nil
}
