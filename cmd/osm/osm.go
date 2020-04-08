package main

import (
	goflag "flag"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var globalUsage = `osm enables you to install and manage the 
open service mesh in your Kubernetes cluster

To install and configure open service mesh, run:

   $ osm install
`

func newRootCmd(args []string, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "osm",
		Short: "Install and manage open service mesh",
		Long:  globalUsage,
	}

	cmd.PersistentFlags().AddGoFlagSet(goflag.CommandLine)
	flags := cmd.PersistentFlags()

	// Add subcommands here
	cmd.AddCommand(
		newInstallCmd(out),
	)

	flags.Parse(args)

	return cmd
}

func main() {
	cmd := newRootCmd(os.Args[1:], os.Stdout)

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func getKubeClient(context string) (kubernetes.Interface, error) {
	kubeConfigPath := filepath.Join(homeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}
