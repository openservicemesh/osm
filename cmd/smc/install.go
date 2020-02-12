package main

import (
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const installDesc = `
This command installs the smc control plane on the Kubernetes cluster.
`

type installCmd struct {
	out        io.Writer
	namespace  string
	kubeClient kubernetes.Interface

	containerRegistry       string
	containerRegistrySecret string
}

func newInstallCmd(out io.Writer) *cobra.Command {

	install := &installCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "install",
		Short: "install smc control plane",
		Long:  installDesc,
		RunE: func(_ *cobra.Command, args []string) error {
			return install.run()
		},
	}

	f := cmd.Flags()
	f.StringVarP(&install.namespace, "namespace", "n", "smc", "namespace to install control plane components")
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
	i.kubeClient = client

	if err := i.deployCDS(); err != nil {
		return err
	}
	if err := i.deploySDS(); err != nil {
		return err
	}
	if err := i.deployEDS(); err != nil {
		return err
	}
	if err := i.deployRDS(); err != nil {
		return err
	}
	return nil
}

func (i *installCmd) deployRDS() error {
	deploymentsClient := i.kubeClient.AppsV1().Deployments(i.namespace)
	servicesClient := i.kubeClient.CoreV1().Services(i.namespace)

	deployment, service := generateRDSKubernetesConfig(i.namespace, i.containerRegistry, i.containerRegistrySecret)
	if _, err := deploymentsClient.Create(deployment); err != nil {
		return err
	}

	if _, err := servicesClient.Create(service); err != nil {
		return err
	}

	return nil
}

func (i *installCmd) deployEDS() error {
	deploymentsClient := i.kubeClient.AppsV1().Deployments(i.namespace)
	servicesClient := i.kubeClient.CoreV1().Services(i.namespace)

	deployment, service := generateEDSKubernetesConfig(i.namespace, i.containerRegistry, i.containerRegistrySecret)

	if _, err := deploymentsClient.Create(deployment); err != nil {
		return err
	}

	if _, err := servicesClient.Create(service); err != nil {
		return err
	}

	return nil
}

func (i *installCmd) deployCDS() error {
	deploymentsClient := i.kubeClient.AppsV1().Deployments(i.namespace)
	servicesClient := i.kubeClient.CoreV1().Services(i.namespace)

	deployment, service := generateCDSKubernetesConfig(i.namespace, i.containerRegistry, i.containerRegistrySecret)

	if _, err := deploymentsClient.Create(deployment); err != nil {
		return err
	}

	if _, err := servicesClient.Create(service); err != nil {
		return err
	}
	return nil
}

func (i *installCmd) deploySDS() error {
	deploymentsClient := i.kubeClient.AppsV1().Deployments(i.namespace)
	servicesClient := i.kubeClient.CoreV1().Services(i.namespace)

	deployment, service := generateSDSKubernetesConfig(i.namespace, i.containerRegistry, i.containerRegistrySecret)
	if _, err := deploymentsClient.Create(deployment); err != nil {
		return err
	}

	if _, err := servicesClient.Create(service); err != nil {
		return err
	}
	return nil
}

func getKubeClient(context string) (kubernetes.Interface, error) {
	kubeConfigPath := filepath.Join(homeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func int32Ptr(i int32) *int32 { return &i }
