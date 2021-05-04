package main

import (
	"context"
	"fmt"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

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
