package metrics

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/open-service-mesh/osm/demo/cmd/common"
	"github.com/open-service-mesh/osm/pkg/constants"
)

// DeployPrometheus deploys various components of prometheus service
func DeployPrometheus(clientSet *kubernetes.Clientset, namespace string) error {
	prometheusSvc := os.Getenv(common.PrometheusVar)
	serviceAccount := fmt.Sprintf("%s-serviceaccount", prometheusSvc)
	prometheusRetentionTime := common.GetEnv(common.PrometheusRetention, constants.PrometheusDefaultRetentionTime)
	if err := deployPrometheusRBAC(clientSet, prometheusSvc, namespace, serviceAccount); err != nil {
		return fmt.Errorf("Unable to deploy prometheus RBAC : %v", err)
	}
	if err := deployPrometheusConfigMap(clientSet, prometheusSvc, namespace); err != nil {
		return fmt.Errorf("Unable to deploy prometheus config map : %v", err)
	}
	if err := deployPrometheusService(clientSet, prometheusSvc, namespace); err != nil {
		return fmt.Errorf("Unable to deploy prometheus service : %v", err)
	}
	if err := deployPrometheusDeployment(clientSet, prometheusSvc, namespace, prometheusRetentionTime); err != nil {
		return fmt.Errorf("Unable to deploy prometheus deployment : %v", err)
	}
	return nil
}

func deployPrometheusService(clientSet *kubernetes.Clientset, svc string, namespace string) error {
	service := generatePrometheusService(svc, namespace)
	if _, err := clientSet.CoreV1().Services(namespace).Create(context.Background(), service, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func deployPrometheusDeployment(clientSet *kubernetes.Clientset, svc string, namespace string, retentionTime string) error {
	deployment := generatePrometheusDeployment(svc, namespace, retentionTime)
	if _, err := clientSet.AppsV1().Deployments(namespace).Create(context.Background(), deployment, metav1.CreateOptions{}); err != nil {
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
	if _, err := clientSet.CoreV1().ConfigMaps(namespace).Create(context.Background(), configMap, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func deployPrometheusRBAC(clientSet *kubernetes.Clientset, svc string, namespace string, serviceAccountName string) error {
	role, roleBinding, serviceAccount := generatePrometheusRBAC(svc, namespace, serviceAccountName)
	if _, err := clientSet.RbacV1().ClusterRoles().Create(context.Background(), role, metav1.CreateOptions{}); err != nil {
		return err
	}
	if _, err := clientSet.RbacV1().ClusterRoleBindings().Create(context.Background(), roleBinding, metav1.CreateOptions{}); err != nil {
		return err
	}
	if _, err := clientSet.CoreV1().ServiceAccounts(namespace).Create(context.Background(), serviceAccount, metav1.CreateOptions{}); err != nil {
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
