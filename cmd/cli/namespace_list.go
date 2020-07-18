package main

import (
	"context"
	"fmt"
	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/cmd/helm/require"
	"io"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"text/tabwriter"
)

const namespaceListDescription = `
This command will list namespace information for all meshes. It is possible to filter by given mesh.
`

type namespaceListCmd struct {
	out       io.Writer
	meshName  string
	clientSet kubernetes.Interface
}

func newNamespaceList(out io.Writer) *cobra.Command {
	namespaceList := &namespaceListCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "list namespaces for meshes",
		Long:  namespaceAddDescription,
		Args:  require.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 1 {
				namespaceList.meshName = args[0]
			}

			config, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return fmt.Errorf("Error fetching kubeconfig")
			}

			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("Could not access Kubernetes cluster. Check kubeconfig")
			}
			namespaceList.clientSet = clientset
			return namespaceList.run()
		},
	}

	//add mesh name flag
	f := cmd.Flags()
	f.StringVar(&namespaceList.meshName, "mesh-name", "", "Name of service mesh to list namespaces")

	return cmd
}

func (l *namespaceListCmd) run() error {
	namespaces, err := l.selectNamespaces()
	if err != nil {
		return fmt.Errorf("Could not list namespaces related to osm [%s]: %v", l.meshName, err)
	}

	if len(namespaces.Items) == 0 {
		if l.meshName != "" {
			fmt.Fprintf(l.out, "No namespaces in mesh [%s]\n", l.meshName)
			return nil
		}

		fmt.Fprintf(l.out, "No namespaces in any mesh\n")
		return nil
	}

	//todo extract this to factory
	w := tabwriter.NewWriter(l.out, 6, 4, 3, ' ', 0)
	fmt.Fprintln(w, "NAMESPACE\tMESH\t")
	for _, ns := range namespaces.Items {
		osmName, _ := ns.ObjectMeta.Labels[constants.OSMKubeResourceMonitorAnnotation]
		fmt.Fprintf(w, "%s\t%s\t\n", ns.Name, osmName)
	}
	w.Flush()

	return nil
}

func (l *namespaceListCmd) selectNamespaces() (*v1.NamespaceList, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	selector := constants.OSMKubeResourceMonitorAnnotation
	if l.meshName != "" {
		selector = fmt.Sprintf("%s=%s", selector, l.meshName)
	}

	return l.clientSet.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
}
