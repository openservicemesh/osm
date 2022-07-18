package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/constants"
)

const meshListDescription = `
This command will list all the osm control planes running in a Kubernetes cluster and controller pods.`

type meshListCmd struct {
	out       io.Writer
	config    *rest.Config
	clientSet kubernetes.Interface
	localPort uint16
}

type meshInfo struct {
	name                string
	namespace           string
	version             string
	monitoredNamespaces []string
}

type meshSmiInfo struct {
	name                 string
	namespace            string
	smiSupportedVersions []string
}

func newMeshList(out io.Writer) *cobra.Command {
	listCmd := &meshListCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "list control planes in k8s cluster",
		Long:  meshListDescription,
		Args:  cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, args []string) error {
			config, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return fmt.Errorf("Error fetching kubeconfig: %w", err)
			}
			listCmd.config = config
			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("Could not access Kubernetes cluster, check kubeconfig: %w", err)
			}
			listCmd.clientSet = clientset
			return listCmd.run()
		},
	}

	f := cmd.Flags()
	f.Uint16VarP(&listCmd.localPort, "local-port", "p", constants.OSMHTTPServerPort, "Local port to use for port forwarding")

	return cmd
}

func (l *meshListCmd) run() error {
	meshInfoList, err := getMeshInfoList(l.config, l.clientSet)
	if err != nil {
		fmt.Fprintf(l.out, "Unable to list meshes within the cluster.\n")
		return err
	}
	if len(meshInfoList) == 0 {
		fmt.Fprintf(l.out, "No osm mesh control planes found\n")
		return nil
	}

	w := newTabWriter(l.out)
	fmt.Fprint(w, getPrettyPrintedMeshInfoList(meshInfoList))
	_ = w.Flush()

	meshSmiInfoList := getSupportedSmiInfoForMeshList(meshInfoList, l.clientSet, l.config, l.localPort)
	fmt.Fprint(w, getPrettyPrintedMeshSmiInfoList(meshSmiInfoList))
	_ = w.Flush()

	fmt.Fprintf(l.out, "\nTo list the OSM controller pods for a mesh, please run the following command passing in the mesh's namespace\n")
	fmt.Fprintf(l.out, "\tkubectl get pods -n <osm-mesh-namespace> -l app=osm-controller\n")

	return nil
}
