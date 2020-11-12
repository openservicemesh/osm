package kube

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Test Announcement Helper Functions", func() {

	Context("Test getPodUID()", func() {
		It("fetches Pod's UID from a Pod Kubernetes object", func() {
			podUID := types.UID("-uid-")
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					UID: podUID,
				},
			}
			actual := getPodUID(pod)
			Expect(actual).To(Equal(podUID))
		})

		It("returns empty UID when no UID", func() {
			pod := &corev1.Pod{}
			actual := getPodUID(pod)
			Expect(actual).To(Equal(types.UID("")))
		})
	})
})
