package main

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	tassert "github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/constants"
)

func TestAnnotateErrorMessageWithActionableMessage(t *testing.T) {
	assert := tassert.New(t)

	type test struct {
		errorMsg     string
		name         string
		namespace    string
		exceptionMsg string
		annotatedMsg string
	}

	actionableMsg := "Use flags to modify the command to suit your needs"

	testCases := []test{
		{
			"Error message with args such as [name: %s], [namespace: %s], and [err: %s]",
			"osm-name",
			"osm-namespace",
			"osm-exception",
			"Error message with args such as [name: osm-name], [namespace: osm-namespace], and [err: osm-exception]\n\n" + actionableMsg,
		},
	}

	for _, tc := range testCases {
		t.Run("Testing annotated error message", func(t *testing.T) {
			assert.Equal(
				tc.annotatedMsg,
				annotateErrorMessageWithActionableMessage(actionableMsg, tc.errorMsg, tc.name, tc.namespace, tc.exceptionMsg).Error())
		})
	}
}

func TestAnnotateErrorMessageWithOsmNamespace(t *testing.T) {
	assert := tassert.New(t)

	type test struct {
		errorMsg     string
		name         string
		namespace    string
		exceptionMsg string
		annotatedMsg string
	}

	osmNamespaceErrorMsg := fmt.Sprintf(
		"Note: The command failed when run in the OSM namespace [%s].\n"+
			"Use the global flag --osm-namespace if [%s] is not the intended OSM namespace.",
		settings.Namespace(), settings.Namespace())

	testCases := []test{
		{
			"Error message with args such as [name: %s], [namespace: %s], and [err: %s]",
			"osm-name",
			"osm-namespace",
			"osm-exception",
			"Error message with args such as [name: osm-name], [namespace: osm-namespace], and [err: osm-exception]\n\n" + osmNamespaceErrorMsg,
		},
	}

	for _, tc := range testCases {
		t.Run("Testing annotated error message", func(t *testing.T) {
			assert.Equal(
				tc.annotatedMsg,
				annotateErrorMessageWithOsmNamespace(tc.errorMsg, tc.name, tc.namespace, tc.exceptionMsg).Error())
		})
	}
}

var _ = Describe("Test getting pretty printed output of a list of meshes", func() {
	var (
		meshInfoList []meshInfo
	)

	Context("empty mesh list", func() {

		meshInfoList = []meshInfo{}
		pp := getPrettyPrintedMeshInfoList(meshInfoList)

		It("should have correct output", func() {
			Expect(pp).To(Equal("\nMESH NAME\tMESH NAMESPACE\tCONTROLLER PODS\tVERSION\tSMI SUPPORTED\tADDED NAMESPACES\n"))
		})
	})

	Context("non-empty mesh list", func() {

		meshInfoList = []meshInfo{
			{
				name:                 "m1",
				namespace:            "ns1",
				controllerPods:       []string{"p1", "p2", "p3"},
				version:              "v1",
				smiSupportedVersions: []string{"s1", "s2", "s3"},
				monitoredNamespaces:  []string{"mn1", "mn2", "mn3"},
			},
			{
				name:                 "m2",
				namespace:            "ns2",
				controllerPods:       []string{"p4", "p5", "p6"},
				version:              "v2",
				smiSupportedVersions: []string{"s4", "s5", "s6"},
				monitoredNamespaces:  []string{"mn4", "mn5", "mn6"},
			},
		}

		It("should have correct output", func() {
			Expect(getPrettyPrintedMeshInfoList(meshInfoList)).To(Equal("\nMESH NAME\tMESH NAMESPACE\tCONTROLLER PODS\tVERSION\tSMI SUPPORTED\tADDED NAMESPACES\nm1\tns1\tp1,p2,p3\tv1\ts1,s2,s3\tmn1,mn2,mn3\nm2\tns2\tp4,p5,p6\tv2\ts4,s5,s6\tmn4,mn5,mn6\n"))
		})

	})
})

var _ = Describe("Test getting mesh controller metadata", func() {
	meshName := "testMesh"
	meshNamespace := "testNamespace"

	Context("mesh controller metadata in clientSet", func() {
		fakeClientSet := fake.NewSimpleClientset()
		_, err := addDeployment(fakeClientSet, "osm-controller", meshName, meshNamespace, "testVersion0.1.2", true)
		Expect(err).NotTo(HaveOccurred())

		It("asd", func() {
			deployment, err := getMeshControllerDeployment(fakeClientSet, meshName, meshNamespace)
			Expect(err).NotTo(HaveOccurred())

			Expect(deployment.ObjectMeta.Namespace).To(Equal(meshNamespace))
			Expect(deployment.ObjectMeta.Labels["meshName"]).To(Equal(meshName))
			Expect(deployment.ObjectMeta.Labels[constants.OSMAppVersionLabelKey]).To(Equal("testVersion0.1.2"))
			Expect(deployment.ObjectMeta.Name).To(Equal("osm-controller"))

		})
	})
})

// helper function for tests that adds deployment to the clientset
func addDeployment(fakeClientSet kubernetes.Interface, depName string, meshName string, namespace string, osmVersion string, isMesh bool) (*v1.Deployment, error) {
	dep := createDeployment(depName, meshName, osmVersion, isMesh)
	return fakeClientSet.AppsV1().Deployments(namespace).Create(context.TODO(), dep, metav1.CreateOptions{})
}

// helper function for tests that creates a deployment for mesh and non-mesh deployments
func createDeployment(deploymentName, meshName string, osmVersion string, isMesh bool) *v1.Deployment {
	labelMap := make(map[string]string)
	if isMesh {
		labelMap["app"] = constants.OSMControllerName
		labelMap["meshName"] = meshName
		labelMap[constants.OSMAppVersionLabelKey] = osmVersion
	} else {
		labelMap["app"] = "non-mesh-app"
	}
	dep := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   deploymentName,
			Labels: labelMap,
		},
	}
	return dep
}
