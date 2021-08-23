package cli

import (
	"github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	corev1 "k8s.io/api/core/v1"
)

const (
	serviceAccountKind = "ServiceAccount"
)

// DoesTargetRefDstPod checks whether the TrafficTarget spec refers to the destination pod's service account
func DoesTargetRefDstPod(spec v1alpha3.TrafficTargetSpec, dstPod *corev1.Pod) bool {
	if spec.Destination.Kind != serviceAccountKind {
		return false
	}

	// Map traffic targets to the given pods
	if spec.Destination.Name == dstPod.Spec.ServiceAccountName && spec.Destination.Namespace == dstPod.Namespace {
		return true
	}
	return false
}

// DoesTargetRefSrcPod checks whether the TrafficTarget spec refers to the source pod's service account
func DoesTargetRefSrcPod(spec v1alpha3.TrafficTargetSpec, srcPod *corev1.Pod) bool {
	for _, source := range spec.Sources {
		if source.Kind != serviceAccountKind {
			continue
		}

		if source.Name == srcPod.Spec.ServiceAccountName && source.Namespace == srcPod.Namespace {
			return true
		}
	}
	return false
}
