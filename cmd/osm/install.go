package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const installDesc = `
This command installs the osm control plane on the Kubernetes cluster.
`
const (
	serviceAccountName = "osm-xds"
	certValidityTime   = 20 * time.Minute
)

type installCmd struct {
	out                     io.Writer
	namespace               string
	containerRegistry       string
	containerRegistrySecret string
	kubeClient              kubernetes.Interface
}

func newInstallCmd(out io.Writer) *cobra.Command {

	install := &installCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "install",
		Short: "install osm control plane",
		Long:  installDesc,
		RunE: func(_ *cobra.Command, args []string) error {
			return install.run()
		},
	}

	f := cmd.Flags()
	f.StringVarP(&install.namespace, "namespace", "n", "osm-system", "namespace to install control plane components")
	f.StringVar(&install.containerRegistry, "container-registry", "smctest.azurecr.io", "container registry that hosts control plane component images")
	f.StringVar(&install.containerRegistrySecret, "container-registry-secret", "acr-creds", "name of Kubernetes secret for container registry credentials to be created if it doesn't already exist")

	return cmd
}

func (i *installCmd) run() error {
	kubeClient, err := getKubeClient("")
	if err != nil {
		return err
	}
	i.kubeClient = kubeClient

	namespaceExists := false
	fmt.Fprintf(i.out, "Creating Kubernetes namespace %s\n", i.namespace)
	_, err = i.kubeClient.CoreV1().Namespaces().Create(context.Background(), generateNamespaceConfig(i.namespace), metav1.CreateOptions{})
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
		namespaceExists = true
		fmt.Fprintf(i.out, "Kubernetes namespace [%s] already exists\n", i.namespace)
	}
	if !namespaceExists {
		fmt.Fprintf(i.out, "Successfully created Kubernetes namespace [%s]\n", i.namespace)
	}

	if i.containerRegistrySecret != "" {
		fmt.Fprintf(i.out, "Creating Kubernetes secret [%s] for container registry [%s] credentials\n",
			i.containerRegistrySecret, i.containerRegistry)
		if err := i.createContainerRegistrySecret(); err != nil {
			return fmt.Errorf("error creating Kubernetes secret [%s] for container registry [%s] credentials: %s",
				i.containerRegistrySecret, i.containerRegistry, err)
		}
		fmt.Fprintf(i.out, "Successfully created Kubernetes secret [%s]\n", i.containerRegistrySecret)
	}

	fmt.Fprintf(i.out, "Setting up CA and generating certificates for osm control plane\n")
	if err = generateAdsSecrets(); err != nil {
		return fmt.Errorf("error setting up CA and generating certificates for osm control plane: %s", err)
	}
	fmt.Fprintf(i.out, "Successfully set up CA and generated certificates for osm control plane\n")

	fmt.Fprintf(i.out, "Generating Kubernetes RBAC for osm control plane\n")
	if err := i.deployRBAC(serviceAccountName); err != nil {
		return fmt.Errorf("error generating Kubernetes RBAC for osm control plane: %s", err)
	}
	fmt.Fprintf(i.out, "Successfully generated Kubernetes RBAC for osm control plane\n")

	fmt.Fprintf(i.out, "Deploying osm control plane components\n")
	if err := i.deploy("ads", serviceAccountName, 15128); err != nil {
		return fmt.Errorf("error deployment osm Kubernetes deployment and service: %s", err)
	}
	fmt.Fprintf(i.out, "Successfully deployed osm Kubernetes deployment and service")

	fmt.Fprintf(i.out, "Generating certificates for sidecar injection webhook\n")
	if err := generateWebhookSecrets(); err != nil {
		return fmt.Errorf("error generating certificates for sidecar injection webhook: %s", err)
	}
	fmt.Fprintf(i.out, "Successfully generated webhook certificates\n")

	//TODO(michelle): wait for ads pod to be ready before deploying webhook config
	fmt.Fprintf(i.out, "Deploying sidecar injection webhook\n")
	if err := i.deployWebhook(); err != nil {
		return fmt.Errorf("error deploying webhook: %s", err)
	}
	fmt.Fprintf(i.out, "Successfully deployed webhook\n")
	fmt.Fprintf(i.out, "Successfully deployed osm control plane\n")
	fmt.Fprintf(i.out, "Happy Meshing!\n")

	return nil
}

func (i *installCmd) createContainerRegistrySecret() error {
	registryName := strings.Split(i.containerRegistry, ".")[0]
	cmd := exec.Command("az", "acr", "credential", "show", "-n", registryName, "--query", "passwords[0].value")
	output, err := cmd.CombinedOutput()
	if err != nil {
		i.out.Write(output)
		return err
	}
	password := strings.Split(string(output), "\"")[1]
	cmd = exec.Command("kubectl", "create", "secret", "docker-registry", i.containerRegistrySecret,
		"-n", i.namespace,
		"--docker-server", i.containerRegistry,
		"--docker-username", registryName,
		"--docker-email", "noone@example.com",
		"--docker-password", password,
	)
	output, err = cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "AlreadyExists") {
			fmt.Fprintf(i.out, "Kubernetes secret [%s] already exists\n", i.containerRegistrySecret)
			//TODO: log that creds already exist
		} else {
			i.out.Write(output)
			return err
		}
	}
	return nil
}

func generateAdsSecrets() error {
	cmd := exec.Command("./demo/gen-ca.sh") //TODO: update to use tresor
	if err := cmd.Run(); err != nil {
		return err
	}
	cmd = exec.Command("./demo/deploy-secrets.sh", "ads") //TODO: update to use tresor
	err := cmd.Run()
	return err

}

func (i *installCmd) deploy(name, serviceAccountName string, port int32) error {
	deployment, service := generateKubernetesConfig(name, i.namespace, serviceAccountName, i.containerRegistry, i.containerRegistrySecret, port)

	_, err := i.kubeClient.AppsV1().Deployments(i.namespace).Create(context.Background(), deployment, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	if _, err := i.kubeClient.CoreV1().Services(i.namespace).Create(context.Background(), service, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

func (i *installCmd) deployRBAC(serviceAccountName string) error {
	role, roleBinding, serviceAccount := generateRBAC(i.namespace, serviceAccountName)
	if _, err := i.kubeClient.RbacV1().ClusterRoles().Create(context.Background(), role, metav1.CreateOptions{}); err != nil {
		return err
	}
	if _, err := i.kubeClient.RbacV1().ClusterRoleBindings().Create(context.Background(), roleBinding, metav1.CreateOptions{}); err != nil {
		return err
	}
	if _, err := i.kubeClient.CoreV1().ServiceAccounts(i.namespace).Create(context.Background(), serviceAccount, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func (i *installCmd) deployWebhook() error {
	caBundle, err := genCABundle()
	if err != nil {
		return err
	}
	webhookConfig := generateWebhookConfig(caBundle, i.namespace)
	_, err = i.kubeClient.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Create(context.Background(), webhookConfig, metav1.CreateOptions{})

	return err
}

func genCABundle() ([]byte, error) {
	data, err := ioutil.ReadFile("./demo/webhook-certs/ca.crt") //TODO: const
	if err != nil {
		return nil, err
	}

	return data, nil
}

func generateWebhookSecrets() error {
	cmd := exec.Command("./demo/deploy-webhook-secrets.sh") //TODO make const
	err := cmd.Run()
	return err
}
