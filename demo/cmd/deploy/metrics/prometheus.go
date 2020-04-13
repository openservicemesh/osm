package metrics

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/open-service-mesh/osm/demo/cmd/common"
	"k8s.io/client-go/kubernetes"
)

// DeployPrometheus deploys various components of prometheus service
func DeployPrometheus(clientSet *kubernetes.Clientset, namespace string) error {
	prometheusSvc := os.Getenv(common.PrometheusVar)
	serviceAccount := fmt.Sprintf("%s-serviceaccount", prometheusSvc)
	if err := deployPrometheusRBAC(clientSet, prometheusSvc, namespace, serviceAccount); err != nil {
		return fmt.Errorf("Unable to deploy prometheus RBAC : %v", err)
	}
	if err := deployPrometheusConfigMap(clientSet, prometheusSvc, namespace); err != nil {
		return fmt.Errorf("Unable to deploy prometheus config map : %v", err)
	}
	if err := deployPrometheusService(clientSet, prometheusSvc, namespace); err != nil {
		return fmt.Errorf("Unable to deploy prometheus service : %v", err)
	}
	if err := deployPrometheusDeployment(clientSet, prometheusSvc, namespace); err != nil {
		return fmt.Errorf("Unable to deploy prometheus deployment : %v", err)
	}
	return nil
}

func deployPrometheusService(clientSet *kubernetes.Clientset, svc string, namespace string) error {
	service := generatePrometheusService(svc, namespace)
	if _, err := clientSet.CoreV1().Services(namespace).Create(service); err != nil {
		return err
	}
	return nil
}

func deployPrometheusDeployment(clientSet *kubernetes.Clientset, svc string, namespace string) error {
	deployment := generatePrometheusDeployment(svc, namespace)
	if _, err := clientSet.AppsV1().Deployments(namespace).Create(deployment); err != nil {
		return err
	}
	return nil
}

func deployPrometheusConfigMap(clientSet *kubernetes.Clientset, svc string, namespace string) error {
	prometheusYaml, err := getPrometheusYamlConfig()
	if err != nil {
		return err
	}
	configMap := generatePrometheusConfigMap(svc, namespace, prometheusYaml)
	if _, err := clientSet.CoreV1().ConfigMaps(namespace).Create(configMap); err != nil {
		return err
	}
	return nil
}

func deployPrometheusRBAC(clientSet *kubernetes.Clientset, svc string, namespace string, serviceAccountName string) error {
	role, roleBinding, serviceAccount := generatePrometheusRBAC(svc, namespace, serviceAccountName)
	if _, err := clientSet.RbacV1().ClusterRoles().Create(role); err != nil {
		return err
	}
	if _, err := clientSet.RbacV1().ClusterRoleBindings().Create(roleBinding); err != nil {
		return err
	}
	if _, err := clientSet.CoreV1().ServiceAccounts(namespace).Create(serviceAccount); err != nil {
		return err
	}
	return nil
}

func getPrometheusYamlConfig() (string, error) {
	var prometheusYaml string
	fileContent, err := ioutil.ReadFile("./demo/cmd/deploy/metrics/prometheus-config.txt")
	if err != nil {
		err = fmt.Errorf("Unable to get prometheus config : %v", err)
		return prometheusYaml, err
	}
	prometheusYaml = string(fileContent)
	return prometheusYaml, nil
}
