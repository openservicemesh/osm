package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/constants"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
)

// confirm displays a prompt `s` to the user and returns a bool indicating yes / no
// If the lowercased, trimmed input begins with anything other than 'y', it returns false
// It accepts an int `tries` representing the number of attempts before returning false
func confirm(stdin io.Reader, stdout io.Writer, s string, tries int) (bool, error) {
	r := bufio.NewReader(stdin)

	for ; tries > 0; tries-- {
		fmt.Fprintf(stdout, "%s [y/n]: ", s)

		res, err := r.ReadString('\n')
		if err != nil {
			return false, err
		}

		// Empty input (i.e. "\n")
		if len(res) < 2 {
			continue
		}

		switch strings.ToLower(strings.TrimSpace(res)) {
		case "y":
			return true, nil
		case "n":
			return false, nil
		default:
			fmt.Fprintf(stdout, "Invalid input.\n")
			continue
		}
	}

	return false, nil
}

// getPrettyPrintedMeshInfoList a pretty printed list of meshes.
// If showIndex is true, then a 1-indexed number is prepended before each mesh.
func getPrettyPrintedMeshInfoList(meshInfoList []meshInfo) string {
	s := "\nMESH NAME\tMESH NAMESPACE\tCONTROLLER PODS\tVERSION\tSMI SUPPORTED\tADDED NAMESPACES\n"

	for _, meshInfo := range meshInfoList {
		m := fmt.Sprintf(
			"%s\t%s\t%s\t%s\t%s\t%s\n",
			meshInfo.name,
			meshInfo.namespace,
			strings.Join(meshInfo.controllerPods, ","),
			meshInfo.version,
			strings.Join(meshInfo.smiSupportedVersions, ","),
			strings.Join(meshInfo.monitoredNamespaces, ","),
		)
		s += m
	}

	return s
}

// getMeshInfoList returns a list of meshes (including the info of each mesh) within the cluster
func getMeshInfoList(restConfig *rest.Config, clientSet kubernetes.Interface, localPort uint16) ([]meshInfo, error) {
	var meshInfoList []meshInfo

	osmControllerDeployments, err := getControllerDeployments(clientSet)
	if err != nil {
		return meshInfoList, errors.Errorf("Could not list deployments %v", err)
	}
	if len(osmControllerDeployments.Items) == 0 {
		return meshInfoList, nil
	}

	for _, osmControllerDeployment := range osmControllerDeployments.Items {
		meshName := osmControllerDeployment.ObjectMeta.Labels["meshName"]
		meshNamespace := osmControllerDeployment.ObjectMeta.Namespace
		meshControllerPods := getNamespacePods(clientSet, meshName, meshNamespace)

		meshVersion := osmControllerDeployment.ObjectMeta.Labels[constants.OSMAppVersionLabelKey]
		if meshVersion == "" {
			meshVersion = "Unknown"
		}

		meshSmiSupportedVersions := []string{"Unknown"}
		if pods, ok := meshControllerPods["Pods"]; ok && len(pods) > 0 {
			smiMap, err := getSupportedSmiForControllerPod(meshControllerPods["Pods"][0], meshNamespace, restConfig, clientSet, localPort)
			if err == nil {
				meshSmiSupportedVersions = []string{}
				for smi, version := range smiMap {
					meshSmiSupportedVersions = append(meshSmiSupportedVersions, fmt.Sprintf("%s:%s", smi, version))
				}
			}
		}

		meshMonitoredNamespaces := []string{}
		nsList, err := selectNamespacesMonitoredByMesh(meshName, clientSet)
		if err == nil && len(nsList.Items) > 0 {
			for _, ns := range nsList.Items {
				meshMonitoredNamespaces = append(meshMonitoredNamespaces, ns.Name)
			}
		}

		meshInfoList = append(meshInfoList, meshInfo{
			name:                 meshName,
			namespace:            meshNamespace,
			controllerPods:       meshControllerPods["Pods"],
			version:              meshVersion,
			smiSupportedVersions: meshSmiSupportedVersions,
			monitoredNamespaces:  meshMonitoredNamespaces,
		})
	}

	return meshInfoList, nil
}

// getNamespacePods returns a map of controller pods
func getNamespacePods(clientSet kubernetes.Interface, m string, ns string) map[string][]string {
	x := make(map[string][]string)

	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"app": constants.OSMControllerName}}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	pods, _ := clientSet.CoreV1().Pods(ns).List(context.TODO(), listOptions)

	for pno := 0; pno < len(pods.Items); pno++ {
		x["Pods"] = append(x["Pods"], pods.Items[pno].GetName())
	}

	return x
}

// getControllerDeployments returns a list of Deployments corresponding to osm-controller
func getControllerDeployments(clientSet kubernetes.Interface) (*v1.DeploymentList, error) {
	deploymentsClient := clientSet.AppsV1().Deployments("") // Get deployments from all namespaces
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"app": constants.OSMControllerName}}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	return deploymentsClient.List(context.TODO(), listOptions)
}

// getMeshNames returns a set of mesh names corresponding to meshes within the cluster
func getMeshNames(clientSet kubernetes.Interface) mapset.Set {
	meshList := mapset.NewSet()

	deploymentList, _ := getControllerDeployments(clientSet)
	for _, elem := range deploymentList.Items {
		meshList.Add(elem.ObjectMeta.Labels["meshName"])
	}

	return meshList
}

func getSupportedSmiForControllerPod(pod string, namespace string, restConfig *rest.Config, clientSet kubernetes.Interface, localPort uint16) (map[string]string, error) {
	dialer, err := k8s.DialerToPod(restConfig, clientSet, pod, namespace)
	if err != nil {
		return nil, err
	}

	portForwarder, err := k8s.NewPortForwarder(dialer, fmt.Sprintf("%d:%d", localPort, constants.OSMHTTPServerPort))
	if err != nil {
		return nil, errors.Errorf("Error setting up port forwarding: %s", err)
	}

	var smiSupported map[string]string

	err = portForwarder.Start(func(pf *k8s.PortForwarder) error {
		defer pf.Stop()
		url := fmt.Sprintf("http://localhost:%d%s", localPort, constants.HTTPServerSmiVersionPath)

		// #nosec G107: Potential HTTP request made with variable url
		resp, err := http.Get(url)
		if err != nil {
			return errors.Errorf("Error fetching url %s: %s", url, err)
		}

		if err := json.NewDecoder(resp.Body).Decode(&smiSupported); err != nil {
			return errors.Errorf("Error rendering HTTP response: %s", err)
		}
		return nil
	})
	if err != nil {
		return nil, errors.Errorf("Error retrieving supported SMI versions for pod %s in namespace %s: %s", pod, namespace, err)
	}

	return smiSupported, nil
}

func annotateErrorMessageWithOsmNamespace(errMsgFormat string, args ...interface{}) error {
	osmNamespaceErrorMsg := fmt.Sprintf(
		"Note: The command failed when run in the OSM namespace [%s].\n"+
			"Use the global flag --osm-namespace if [%s] is not the intended OSM namespace.",
		settings.Namespace(), settings.Namespace())

	return annotateErrorMessageWithActionableMessage(osmNamespaceErrorMsg, errMsgFormat, args...)
}

func annotateErrorMessageWithActionableMessage(actionableMessage string, errMsgFormat string, args ...interface{}) error {
	if !strings.HasSuffix(errMsgFormat, "\n") {
		errMsgFormat += "\n"
	}

	if !strings.HasSuffix(errMsgFormat, "\n\n") {
		errMsgFormat += "\n"
	}

	return errors.Errorf(errMsgFormat+actionableMessage, args...)
}
