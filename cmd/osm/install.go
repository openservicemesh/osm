package main

import (
	"io"
	"io/ioutil"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
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
	f.StringVar(&install.containerRegistrySecret, "container-registry-secret", "acr-creds", "name of the container registry Kubernetes Secret that contains container registry credentials")

	return cmd
}

func (i *installCmd) run() error {
	context := "" //TOOD: make context flag
	client, err := getKubeClient(context)
	if err != nil {
		return err
	}
	//TODO create namespace if it doesn't already exist
	i.kubeClient = client

	err = generateAdsSecrets()
	if err != nil {
		return err
	}
	if err := i.deployRBAC(serviceAccountName); err != nil {
		return err
	}

	if err := i.deploy("ads", serviceAccountName, 15128); err != nil {
		return err
	}

	if err := generateWebhookSecrets(); err != nil {
		return err
	}

	//TODO(michelle): wait for ads pod to be ready before deploying webhook config
	if err := i.deployWebhook(); err != nil {
	}

	return nil
}

func generateAdsSecrets() error {
	cmd := exec.Command("./demo/gen-ca.sh")
	if err := cmd.Run(); err != nil {
		return err
	}
	cmd = exec.Command("./demo/deploy-secrets.sh", "ads")
	err := cmd.Run()
	return err

}

func (i *installCmd) deploy(name, serviceAccountName string, port int32) error {
	deployment, service := generateKubernetesConfig(name, i.namespace, serviceAccountName, i.containerRegistry, i.containerRegistrySecret, port)

	_, err := i.kubeClient.AppsV1().Deployments(i.namespace).Create(deployment)
	if err != nil {
		return err
	}

	if _, err := i.kubeClient.CoreV1().Services(i.namespace).Create(service); err != nil {
		return err
	}

	return nil
}

func (i *installCmd) deployRBAC(serviceAccountName string) error {
	role, roleBinding, serviceAccount := generateRBAC(i.namespace, serviceAccountName)
	if _, err := i.kubeClient.RbacV1().ClusterRoles().Create(role); err != nil {
		return err
	}
	if _, err := i.kubeClient.RbacV1().ClusterRoleBindings().Create(roleBinding); err != nil {
		return err
	}
	if _, err := i.kubeClient.CoreV1().ServiceAccounts(i.namespace).Create(serviceAccount); err != nil {
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
	_, err = i.kubeClient.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Create(webhookConfig)

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
