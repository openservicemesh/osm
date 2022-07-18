package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	mapset "github.com/deckarep/golang-set"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/k8s"
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

// getPrettyPrintedMeshInfoList returns a pretty printed list of meshes.
func getPrettyPrintedMeshInfoList(meshInfoList []meshInfo) string {
	s := "\nMESH NAME\tMESH NAMESPACE\tVERSION\tADDED NAMESPACES\n"

	for _, meshInfo := range meshInfoList {
		m := fmt.Sprintf(
			"%s\t%s\t%s\t%s\n",
			meshInfo.name,
			meshInfo.namespace,
			meshInfo.version,
			strings.Join(meshInfo.monitoredNamespaces, ","),
		)
		s += m
	}

	return s
}

// getMeshInfoList returns a list of meshes (including the info of each mesh) within the cluster
func getMeshInfoList(restConfig *rest.Config, clientSet kubernetes.Interface) ([]meshInfo, error) {
	var meshInfoList []meshInfo

	osmControllerDeployments, err := getControllerDeployments(clientSet)
	if err != nil {
		return meshInfoList, fmt.Errorf("Could not list deployments %w", err)
	}
	if len(osmControllerDeployments.Items) == 0 {
		return meshInfoList, nil
	}

	for _, osmControllerDeployment := range osmControllerDeployments.Items {
		meshName := osmControllerDeployment.ObjectMeta.Labels["meshName"]
		meshNamespace := osmControllerDeployment.ObjectMeta.Namespace

		meshVersion := osmControllerDeployment.ObjectMeta.Labels[constants.OSMAppVersionLabelKey]
		if meshVersion == "" {
			meshVersion = "Unknown"
		}

		var meshMonitoredNamespaces []string
		nsList, err := selectNamespacesMonitoredByMesh(meshName, clientSet)
		if err == nil && len(nsList.Items) > 0 {
			for _, ns := range nsList.Items {
				meshMonitoredNamespaces = append(meshMonitoredNamespaces, ns.Name)
			}
		}

		meshInfoList = append(meshInfoList, meshInfo{
			name:                meshName,
			namespace:           meshNamespace,
			version:             meshVersion,
			monitoredNamespaces: meshMonitoredNamespaces,
		})
	}

	return meshInfoList, nil
}

// getControllerDeployments returns a list of Deployments corresponding to osm-controller
func getControllerDeployments(clientSet kubernetes.Interface) (*appsv1.DeploymentList, error) {
	deploymentsClient := clientSet.AppsV1().Deployments("") // Get deployments from all namespaces
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{constants.AppLabel: constants.OSMControllerName}}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	return deploymentsClient.List(context.TODO(), listOptions)
}

// getControllerPods returns a list of osm-controller Pods in a specified namespace
func getControllerPods(clientSet kubernetes.Interface, namespace string) (*corev1.PodList, error) {
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{constants.AppLabel: constants.OSMControllerName}}
	podClient := clientSet.CoreV1().Pods(namespace)
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	return podClient.List(context.TODO(), metav1.ListOptions{LabelSelector: listOptions.LabelSelector})
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

// getPrettyPrintedMeshSmiInfoList returns a pretty printed list
// of meshes with supported smi versions
func getPrettyPrintedMeshSmiInfoList(meshSmiInfoList []meshSmiInfo) string {
	s := "\nMESH NAME\tMESH NAMESPACE\tSMI SUPPORTED\n"

	for _, mesh := range meshSmiInfoList {
		m := fmt.Sprintf(
			"%s\t%s\t%s\n",
			mesh.name,
			mesh.namespace,
			strings.Join(mesh.smiSupportedVersions, ","),
		)
		s += m
	}

	return s
}

// getSupportedSmiInfoForMeshList returns a meshSmiInfo list showing
// the supported smi versions for each osm mesh in the mesh list
func getSupportedSmiInfoForMeshList(meshInfoList []meshInfo, clientSet kubernetes.Interface, config *rest.Config, localPort uint16) []meshSmiInfo {
	var meshSmiInfoList []meshSmiInfo

	for _, mesh := range meshInfoList {
		meshControllerPods := k8s.GetOSMControllerPods(clientSet, mesh.namespace)

		meshSmiSupportedVersions := []string{"Unknown"}
		if len(meshControllerPods.Items) > 0 {
			// for listing mesh information, checking info using the first osm-controller pod should suffice
			controllerPod := meshControllerPods.Items[0]
			smiMap, err := getSupportedSmiForControllerPod(controllerPod.Name, mesh.namespace, config, clientSet, localPort)
			if err == nil {
				meshSmiSupportedVersions = []string{}
				for smi, version := range smiMap {
					meshSmiSupportedVersions = append(meshSmiSupportedVersions, fmt.Sprintf("%s:%s", smi, version))
				}
			}
		}
		sort.Strings(meshSmiSupportedVersions)

		meshSmiInfoList = append(meshSmiInfoList, meshSmiInfo{
			name:                 mesh.name,
			namespace:            mesh.namespace,
			smiSupportedVersions: meshSmiSupportedVersions,
		})
	}

	return meshSmiInfoList
}

// getSupportedSmiForControllerPod returns the supported smi versions
// for a given osm controller pod in a namespace
func getSupportedSmiForControllerPod(pod string, namespace string, restConfig *rest.Config, clientSet kubernetes.Interface, localPort uint16) (map[string]string, error) {
	dialer, err := k8s.DialerToPod(restConfig, clientSet, pod, namespace)
	if err != nil {
		return nil, err
	}

	portForwarder, err := k8s.NewPortForwarder(dialer, fmt.Sprintf("%d:%d", localPort, constants.OSMHTTPServerPort))
	if err != nil {
		return nil, fmt.Errorf("Error setting up port forwarding: %w", err)
	}

	var smiSupported map[string]string

	err = portForwarder.Start(func(pf *k8s.PortForwarder) error {
		defer pf.Stop()
		url := fmt.Sprintf("http://localhost:%d%s", localPort, constants.OSMControllerSMIVersionPath)

		// #nosec G107: Potential HTTP request made with variable url
		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("Error fetching url %s: %s", url, err)
		}

		if err := json.NewDecoder(resp.Body).Decode(&smiSupported); err != nil {
			return fmt.Errorf("Error rendering HTTP response: %s", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("Error retrieving supported SMI versions for pod %s in namespace %s: %s", pod, namespace, err)
	}

	for smiAPI, smiAPIVersion := range smiSupported {
		// smiApi looks like HTTPRouteGroup
		// smiApiVersion looks like specs.smi-spec.io/v1alpha4
		// leave out the API group and only keep the version after "/"
		splitVersionInfo := strings.SplitN(smiAPIVersion, "/", 2)
		if len(splitVersionInfo) >= 2 {
			smiSupported[smiAPI] = splitVersionInfo[1]
		}
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

	return fmt.Errorf(errMsgFormat+actionableMessage, args...)
}
