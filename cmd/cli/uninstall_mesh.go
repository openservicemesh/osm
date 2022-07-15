package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	helmStorage "helm.sh/helm/v3/pkg/storage/driver"
	extensionsClientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	k8sApiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/constants"
)

const uninstallMeshDescription = `
This command will uninstall an instance of the osm control plane
given the mesh name and namespace.

Uninstalling OSM will:
(1) remove osm control plane components
(2) remove/un-patch the conversion webhook fields from all the CRDs
(which OSM adds to support multiple CR versions)

The command will not delete:
(1) the namespace the mesh was installed in unless specified via the
--delete-namespace flag.
(2) the cluster-wide resources (i.e. CRDs, mutating and validating webhooks and
secrets) unless specified via the --delete-cluster-wide-resources (or -a) flag

Be careful when using this command as it is destructive and will
disrupt traffic to applications left running with sidecar proxies.
`

type uninstallMeshCmd struct {
	out                        io.Writer
	in                         io.Reader
	config                     *rest.Config
	meshName                   string
	meshNamespace              string
	caBundleSecretName         string
	force                      bool
	deleteNamespace            bool
	client                     *action.Uninstall
	clientSet                  kubernetes.Interface
	localPort                  uint16
	deleteClusterWideResources bool
	extensionsClientset        extensionsClientset.Interface
	actionConfig               *action.Configuration
}

func newUninstallMeshCmd(config *action.Configuration, in io.Reader, out io.Writer) *cobra.Command {
	uninstall := &uninstallMeshCmd{
		out: out,
		in:  in,
	}

	cmd := &cobra.Command{
		Use:   "mesh",
		Short: "uninstall osm control plane instance",
		Long:  uninstallMeshDescription,
		Args:  cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, args []string) error {
			uninstall.actionConfig = config
			uninstall.client = action.NewUninstall(config)

			// get kubeconfig and initialize k8s client
			kubeconfig, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return fmt.Errorf("Error fetching kubeconfig: %w", err)
			}
			uninstall.config = kubeconfig

			uninstall.clientSet, err = kubernetes.NewForConfig(kubeconfig)
			if err != nil {
				return fmt.Errorf("Could not access Kubernetes cluster, check kubeconfig: %w", err)
			}

			uninstall.extensionsClientset, err = extensionsClientset.NewForConfig(kubeconfig)
			if err != nil {
				return fmt.Errorf("Could not access extension client set: %w", err)
			}

			uninstall.meshNamespace = settings.Namespace()
			return uninstall.run()
		},
	}

	f := cmd.Flags()
	f.StringVar(&uninstall.meshName, "mesh-name", "", "Name of the service mesh")
	f.BoolVarP(&uninstall.force, "force", "f", false, "Attempt to uninstall the osm control plane instance without prompting for confirmation.")
	f.BoolVarP(&uninstall.deleteClusterWideResources, "delete-cluster-wide-resources", "a", false, "Cluster wide resources (such as osm CRDs, mutating webhook configurations, validating webhook configurations and osm secrets) are fully deleted from the cluster after control plane components are deleted.")
	f.BoolVar(&uninstall.deleteNamespace, "delete-namespace", false, "Attempt to delete the namespace after control plane components are deleted")
	f.Uint16VarP(&uninstall.localPort, "local-port", "p", constants.OSMHTTPServerPort, "Local port to use for port forwarding")
	f.StringVar(&uninstall.caBundleSecretName, "ca-bundle-secret-name", constants.DefaultCABundleSecretName, "Name of the secret for the OSM CA bundle")

	return cmd
}

func (d *uninstallMeshCmd) run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	meshesToUninstall := []meshInfo{}

	if !settings.IsManaged() {
		meshInfoList, err := getMeshInfoList(d.config, d.clientSet)
		if err != nil {
			return fmt.Errorf("unable to list meshes within the cluster: %w", err)
		}
		if len(meshInfoList) == 0 {
			fmt.Fprintf(d.out, "No OSM control planes found\n")
			return nil
		}

		if d.meshSpecified() {
			// Searches for the mesh specified by the mesh-name flag if specified
			specifiedMeshFound := d.findSpecifiedMesh(meshInfoList)
			if !specifiedMeshFound {
				return nil
			}
		}

		// Adds the mesh to be force uninstalled
		if d.force {
			// For force uninstall, if single mesh in cluster, set default to that mesh
			if len(meshInfoList) == 1 {
				d.meshName = meshInfoList[0].name
				d.meshNamespace = meshInfoList[0].namespace
			}
			forceMesh := meshInfo{name: d.meshName, namespace: d.meshNamespace}
			meshesToUninstall = append(meshesToUninstall, forceMesh)
		} else {
			// print a list of meshes within the cluster for a better user experience
			err := d.printMeshes()
			if err != nil {
				return err
			}
			// Prompts user on whether to uninstall each OSM mesh in the cluster
			uninstallMeshes, err := d.promptMeshUninstall(meshInfoList, meshesToUninstall)
			if err != nil {
				return err
			}
			meshesToUninstall = append(meshesToUninstall, uninstallMeshes...)
		}

		for _, m := range meshesToUninstall {
			// Re-initializes uninstall config with the namespace of the mesh to be uninstalled
			err := d.actionConfig.Init(settings.RESTClientGetter(), m.namespace, "secret", debug)
			if err != nil {
				return err
			}

			_, err = d.client.Run(m.name)
			if err != nil {
				if errors.Is(err, helmStorage.ErrReleaseNotFound) {
					fmt.Fprintf(d.out, "No OSM control plane with mesh name [%s] found in namespace [%s]\n", m.name, m.namespace)
				}

				if !d.deleteClusterWideResources && !d.deleteNamespace {
					return err
				}

				fmt.Fprintf(d.out, "Could not uninstall mesh name [%s] in namespace [%s]- %v - continuing to deleteClusterWideResources and/or deleteNamespace\n", m.name, m.namespace, err)
			}

			if err == nil {
				fmt.Fprintf(d.out, "OSM [mesh name: %s] in namespace [%s] uninstalled\n", m.name, m.namespace)
			}

			err = d.deleteNs(ctx, m.namespace)
			if err != nil {
				return err
			}
		}
	} else {
		fmt.Fprintf(d.out, "OSM CANNOT be uninstalled in a managed environment\n")
		if d.deleteNamespace {
			fmt.Fprintf(d.out, "OSM namespace CANNOT be deleted in a managed environment\n")
		}
	}

	err := d.deleteClusterResources()
	return err
}

func (d *uninstallMeshCmd) meshSpecified() bool {
	return d.meshName != ""
}

func (d *uninstallMeshCmd) findSpecifiedMesh(meshInfoList []meshInfo) bool {
	specifiedMeshFound := d.findMesh(meshInfoList)
	if !specifiedMeshFound {
		fmt.Fprintf(d.out, "Did not find mesh [%s] in namespace [%s]\n", d.meshName, d.meshNamespace)
		// print a list of meshes within the cluster for a better user experience
		if err := d.printMeshes(); err != nil {
			fmt.Fprintf(d.out, "Unable to list meshes in the cluster - [%v]", err)
		}
	}

	return specifiedMeshFound
}

func (d *uninstallMeshCmd) promptMeshUninstall(meshInfoList, meshesToUninstall []meshInfo) ([]meshInfo, error) {
	for _, mesh := range meshInfoList {
		// Only prompt for specified mesh if `mesh-name` is specified
		if d.meshSpecified() && mesh.name != d.meshName {
			continue
		}
		confirm, err := confirm(d.in, d.out, fmt.Sprintf("\nUninstall OSM [mesh name: %s] in namespace [%s] and/or OSM resources?", mesh.name, mesh.namespace), 3)
		if err != nil {
			return nil, err
		}
		if confirm {
			meshesToUninstall = append(meshesToUninstall, mesh)
		}
	}
	return meshesToUninstall, nil
}

func (d *uninstallMeshCmd) deleteNs(ctx context.Context, ns string) error {
	if !d.deleteNamespace {
		return nil
	}
	if err := d.clientSet.CoreV1().Namespaces().Delete(ctx, ns, v1.DeleteOptions{}); err != nil {
		if k8sApiErrors.IsNotFound(err) {
			fmt.Fprintf(d.out, "OSM namespace [%s] not found\n", ns)
			return nil
		}
		return fmt.Errorf("Could not delete OSM namespace [%s] - %v", ns, err)
	}
	fmt.Fprintf(d.out, "OSM namespace [%s] deleted successfully\n", ns)
	return nil
}

func (d *uninstallMeshCmd) deleteClusterResources() error {
	if d.deleteClusterWideResources {
		meshInfoList, err := getMeshInfoList(d.config, d.clientSet)
		if err != nil {
			return fmt.Errorf("unable to list meshes within the cluster: %w", err)
		}
		if len(meshInfoList) != 0 {
			fmt.Fprintf(d.out, "Deleting cluster resources will affect current mesh(es) in cluster:\n")
			for _, m := range meshInfoList {
				fmt.Fprintf(d.out, "[%s] mesh in namespace [%s]\n", m.name, m.namespace)
			}
		}

		failedDeletions := d.uninstallClusterResources()
		if len(failedDeletions) != 0 {
			return fmt.Errorf("Failed to completely delete the following OSM resource types: %+v", failedDeletions)
		}
	}
	return nil
}

// uninstallClusterResources uninstalls all osm and smi-related cluster resources
func (d *uninstallMeshCmd) uninstallClusterResources() []string {
	var failedDeletions []string
	err := d.uninstallCustomResourceDefinitions()
	if err != nil {
		failedDeletions = append(failedDeletions, "CustomResourceDefinitions")
	}

	err = d.uninstallMutatingWebhookConfigurations()
	if err != nil {
		failedDeletions = append(failedDeletions, "MutatingWebhookConfigurations")
	}

	err = d.uninstallValidatingWebhookConfigurations()
	if err != nil {
		failedDeletions = append(failedDeletions, "ValidatingWebhookConfigurations")
	}

	err = d.uninstallSecrets()
	if err != nil {
		failedDeletions = append(failedDeletions, "Secrets")
	}
	return failedDeletions
}

// uninstallCustomResourceDefinitions uninstalls osm and smi-related crds from the cluster.
func (d *uninstallMeshCmd) uninstallCustomResourceDefinitions() error {
	crds := []string{
		"egresses.policy.openservicemesh.io",
		"ingressbackends.policy.openservicemesh.io",
		"meshconfigs.config.openservicemesh.io",
		"meshRootCertificate.config.openservicemesh.io",
		"upstreamtrafficsettings.policy.openservicemesh.io",
		"retries.policy.openservicemesh.io",
		"httproutegroups.specs.smi-spec.io",
		"tcproutes.specs.smi-spec.io",
		"trafficsplits.split.smi-spec.io",
		"traffictargets.access.smi-spec.io",
	}

	var failedDeletions []string
	for _, crd := range crds {
		err := d.extensionsClientset.ApiextensionsV1().CustomResourceDefinitions().Delete(context.Background(), crd, metav1.DeleteOptions{})

		if err == nil {
			fmt.Fprintf(d.out, "Successfully deleted OSM CRD: %s\n", crd)
			continue
		}

		if k8sApiErrors.IsNotFound(err) {
			fmt.Fprintf(d.out, "Ignoring - did not find OSM CRD: %s\n", crd)
		} else {
			fmt.Fprintf(d.out, "Failed to delete OSM CRD %s: %s\n", crd, err.Error())
			failedDeletions = append(failedDeletions, crd)
		}
	}

	if len(failedDeletions) != 0 {
		return fmt.Errorf("Failed to delete the following OSM CRDs: %+v", failedDeletions)
	}

	return nil
}

// uninstallMutatingWebhookConfigurations uninstalls osm-related mutating webhook configurations from the cluster.
func (d *uninstallMeshCmd) uninstallMutatingWebhookConfigurations() error {
	// These label selectors should always match the Helm post-delete hook at charts/osm/templates/cleanup-hook.yaml.
	webhookConfigurationsLabelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
			constants.OSMAppInstanceLabelKey: d.meshName,
			constants.AppLabel:               constants.OSMInjectorName,
		},
	}

	webhookConfigurationsListOptions := metav1.ListOptions{
		LabelSelector: labels.Set(webhookConfigurationsLabelSelector.MatchLabels).String(),
	}

	mutatingWebhookConfigurations, err := d.clientSet.AdmissionregistrationV1().MutatingWebhookConfigurations().List(context.Background(), webhookConfigurationsListOptions)

	if err != nil {
		errMsg := fmt.Sprintf("Failed to list OSM MutatingWebhookConfigurations in the cluster: %s", err.Error())
		fmt.Fprintln(d.out, errMsg)
		return errors.New(errMsg)
	}

	if len(mutatingWebhookConfigurations.Items) == 0 {
		fmt.Fprint(d.out, "Ignoring - did not find any OSM MutatingWebhookConfigurations in the cluster. Use --mesh-name to delete MutatingWebhookConfigurations belonging to a specific mesh if desired\n")
		return nil
	}

	var failedDeletions []string
	for _, mutatingWebhookConfiguration := range mutatingWebhookConfigurations.Items {
		err := d.clientSet.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(context.Background(), mutatingWebhookConfiguration.Name, metav1.DeleteOptions{})

		if err == nil {
			fmt.Fprintf(d.out, "Successfully deleted OSM MutatingWebhookConfiguration: %s\n", mutatingWebhookConfiguration.Name)
		} else {
			fmt.Fprintf(d.out, "Found but failed to delete OSM MutatingWebhookConfiguration %s: %s\n", mutatingWebhookConfiguration.Name, err.Error())
			failedDeletions = append(failedDeletions, mutatingWebhookConfiguration.Name)
		}
	}

	if len(failedDeletions) != 0 {
		return fmt.Errorf("Found but failed to delete the following OSM MutatingWebhookConfigurations: %+v", failedDeletions)
	}

	return nil
}

// uninstallValidatingWebhookConfigurations uninstalls osm-related validating webhook configurations from the cluster.
func (d *uninstallMeshCmd) uninstallValidatingWebhookConfigurations() error {
	// These label selectors should always match the Helm post-delete hook at charts/osm/templates/cleanup-hook.yaml.
	webhookConfigurationsLabelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
			constants.OSMAppInstanceLabelKey: d.meshName,
			constants.AppLabel:               constants.OSMControllerName,
		},
	}

	webhookConfigurationsListOptions := metav1.ListOptions{
		LabelSelector: labels.Set(webhookConfigurationsLabelSelector.MatchLabels).String(),
	}

	validatingWebhookConfigurations, err := d.clientSet.AdmissionregistrationV1().ValidatingWebhookConfigurations().List(context.Background(), webhookConfigurationsListOptions)

	if err != nil {
		errMsg := fmt.Sprintf("Failed to list OSM ValidatingWebhookConfigurations in the cluster: %s", err.Error())
		fmt.Fprintln(d.out, errMsg)
		return errors.New(errMsg)
	}

	if len(validatingWebhookConfigurations.Items) == 0 {
		fmt.Fprint(d.out, "Ignoring - did not find any OSM ValidatingWebhookConfigurations in the cluster. Use --mesh-name to delete ValidatingWebhookConfigurations belonging to a specific mesh if desired\n")
		return nil
	}

	var failedDeletions []string
	for _, validatingWebhookConfiguration := range validatingWebhookConfigurations.Items {
		err := d.clientSet.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(context.Background(), validatingWebhookConfiguration.Name, metav1.DeleteOptions{})

		if err == nil {
			fmt.Fprintf(d.out, "Successfully deleted OSM ValidatingWebhookConfiguration: %s\n", validatingWebhookConfiguration.Name)
			continue
		} else {
			fmt.Fprintf(d.out, "Found but failed to delete OSM ValidatingWebhookConfiguration %s: %s\n", validatingWebhookConfiguration.Name, err.Error())
			failedDeletions = append(failedDeletions, validatingWebhookConfiguration.Name)
		}
	}

	if len(failedDeletions) != 0 {
		return fmt.Errorf("Found but failed to delete the following OSM ValidatingWebhookConfigurations: %+v", failedDeletions)
	}

	return nil
}

// uninstallSecrets uninstalls osm-related secrets from the cluster.
func (d *uninstallMeshCmd) uninstallSecrets() error {
	secrets := []string{
		d.caBundleSecretName,
	}

	var failedDeletions []string
	for _, secret := range secrets {
		err := d.clientSet.CoreV1().Secrets(d.meshNamespace).Delete(context.Background(), secret, metav1.DeleteOptions{})

		if err == nil {
			fmt.Fprintf(d.out, "Successfully deleted OSM secret %s in namespace %s\n", secret, d.meshNamespace)
			continue
		}

		if k8sApiErrors.IsNotFound(err) {
			if secret == d.caBundleSecretName {
				fmt.Fprintf(d.out, "Ignoring - did not find OSM CA bundle secret %s in namespace %s. Use --ca-bundle-secret-name and --osm-namespace to delete a specific mesh namespace's CA bundle secret if desired\n", secret, d.meshNamespace)
			} else {
				fmt.Fprintf(d.out, "Ignoring - did not find OSM secret %s in namespace %s. Use --osm-namespace to delete a specific mesh namespace's secret if desired\n", secret, d.meshNamespace)
			}
		} else {
			fmt.Fprintf(d.out, "Found but failed to delete the OSM secret %s in namespace %s: %s\n", secret, d.meshNamespace, err.Error())
			failedDeletions = append(failedDeletions, secret)
		}
	}

	if len(failedDeletions) != 0 {
		return fmt.Errorf("Found but failed to delete the following OSM secrets in namespace %s: %+v", d.meshNamespace, failedDeletions)
	}

	return nil
}

// findMesh looks for specified `mesh-name` mesh from the meshes in the cluster
func (d *uninstallMeshCmd) findMesh(meshInfoList []meshInfo) bool {
	found := false
	for _, m := range meshInfoList {
		if m.name == d.meshName {
			found = true
			break
		}
	}
	return found
}

// printMeshes prints list of meshes within the cluster for a better user experience
func (d *uninstallMeshCmd) printMeshes() error {
	fmt.Fprintf(d.out, "List of meshes present in the cluster:\n")

	listCmd := &meshListCmd{
		out:       d.out,
		config:    d.config,
		clientSet: d.clientSet,
		localPort: d.localPort,
	}

	err := listCmd.run()
	// Unable to list meshes in the cluster
	if err != nil {
		return err
	}
	return nil
}
