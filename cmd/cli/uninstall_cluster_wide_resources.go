package main

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	extensionsClientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	k8sApiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/constants"
)

const uninstallClusterWideResourcesDescription = `
This command uninstalls cluster-wide resources from the cluster.

This command ensures that cluster-wide resources (such as osm CRDs,
mutating webhook configurations, validating webhook configurations,
and osm secrets) are fully deleted from the cluster.

Note: this command does not delete the osm control plane namespace.

Be careful when using this command as it is potentially destructive.
`

type uninstallClusterWideResourcesCmd struct {
	in                  io.Reader
	out                 io.Writer
	config              *rest.Config
	force               bool
	meshName            string
	meshNamespace       string
	caBundleSecretName  string
	clientset           kubernetes.Interface
	extensionsClientset extensionsClientset.Interface
}

func newUninstallClusterWideResourcesCmd(in io.Reader, out io.Writer) *cobra.Command {
	uninstall := &uninstallClusterWideResourcesCmd{
		in:  in,
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "cluster-wide-resources",
		Short: "uninstall osm cluster-wide resources",
		Long:  uninstallClusterWideResourcesDescription,
		Args:  cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, args []string) error {
			// get kubeconfig and initialize k8s client
			kubeconfig, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return errors.Errorf("Error fetching kubeconfig: %s", err)
			}
			uninstall.config = kubeconfig

			uninstall.clientset, err = kubernetes.NewForConfig(kubeconfig)
			if err != nil {
				return errors.Errorf("Could not access Kubernetes cluster, check kubeconfig: %s", err)
			}

			uninstall.extensionsClientset, err = extensionsClientset.NewForConfig(kubeconfig)
			if err != nil {
				return errors.Errorf("Could not access Kubernetes cluster, check kubeconfig: %s", err)
			}

			uninstall.meshNamespace = settings.Namespace()

			return uninstall.run()
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&uninstall.force, "force", "f", false, "Attempt to uninstall OSM resources without prompting for confirmation.")
	f.StringVar(&uninstall.meshName, "mesh-name", defaultMeshName, "Name of the service mesh")
	f.StringVar(&uninstall.caBundleSecretName, "ca-bundle-secret-name", constants.DefaultCABundleSecretName, "Name of the secret for the OSM CA bundle")

	return cmd
}

func (d *uninstallClusterWideResourcesCmd) run() error {
	err := d.validateCLIParams()
	if err != nil {
		return err
	}

	if !d.force {
		confirm, err := confirm(d.in, d.out, fmt.Sprintf("\nUninstall OSM resources belonging to mesh '%s' in mesh namespace '%s'?", d.meshName, d.meshNamespace), 3)
		if !confirm || err != nil {
			return err
		}
	}

	var failedDeletions []string

	err = d.uninstallCustomResourceDefinitions()
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

	if len(failedDeletions) != 0 {
		return errors.Errorf("Failed to completely delete the following OSM resource types: %+v", failedDeletions)
	}

	return nil
}

// validateCLIParams contains all checks necessary to ensure that various permutations of the CLI flags are consistent
func (d *uninstallClusterWideResourcesCmd) validateCLIParams() error {
	if d.meshName == "" {
		return errors.New("Please specify the mesh name using --mesh-name")
	}
	if d.meshNamespace == "" {
		return errors.New("Please specify the OSM control plane namespace using --osm-namespace")
	}
	if d.caBundleSecretName == "" {
		return errors.Errorf("Please specify the CA bundle secret name using --ca-bundle-secret-name")
	}

	return nil
}

// uninstallCustomResourceDefinitions uninstalls osm and smi-related crds from the cluster.
func (d *uninstallClusterWideResourcesCmd) uninstallCustomResourceDefinitions() error {
	crds := []string{
		"egresses.policy.openservicemesh.io",
		"ingressbackends.policy.openservicemesh.io",
		"meshconfigs.config.openservicemesh.io",
		"multiclusterservices.config.openservicemesh.io",
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
		return errors.Errorf("Failed to delete the following OSM CRDs: %+v", failedDeletions)
	}

	return nil
}

// uninstallMutatingWebhookConfigurations uninstalls osm-related mutating webhook configurations from the cluster.
func (d *uninstallClusterWideResourcesCmd) uninstallMutatingWebhookConfigurations() error {
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

	mutatingWebhookConfigurations, err := d.clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().List(context.Background(), webhookConfigurationsListOptions)

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
		err := d.clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(context.Background(), mutatingWebhookConfiguration.Name, metav1.DeleteOptions{})

		if err == nil {
			fmt.Fprintf(d.out, "Successfully deleted OSM MutatingWebhookConfiguration: %s\n", mutatingWebhookConfiguration.Name)
		} else {
			fmt.Fprintf(d.out, "Found but failed to delete OSM MutatingWebhookConfiguration %s: %s\n", mutatingWebhookConfiguration.Name, err.Error())
			failedDeletions = append(failedDeletions, mutatingWebhookConfiguration.Name)
		}
	}

	if len(failedDeletions) != 0 {
		return errors.Errorf("Found but failed to delete the following OSM MutatingWebhookConfigurations: %+v", failedDeletions)
	}

	return nil
}

// uninstallValidatingWebhookConfigurations uninstalls osm-related validating webhook configurations from the cluster.
func (d *uninstallClusterWideResourcesCmd) uninstallValidatingWebhookConfigurations() error {
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

	validatingWebhookConfigurations, err := d.clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().List(context.Background(), webhookConfigurationsListOptions)

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
		err := d.clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(context.Background(), validatingWebhookConfiguration.Name, metav1.DeleteOptions{})

		if err == nil {
			fmt.Fprintf(d.out, "Successfully deleted OSM ValidatingWebhookConfiguration: %s\n", validatingWebhookConfiguration.Name)
			continue
		} else {
			fmt.Fprintf(d.out, "Found but failed to delete OSM ValidatingWebhookConfiguration %s: %s\n", validatingWebhookConfiguration.Name, err.Error())
			failedDeletions = append(failedDeletions, validatingWebhookConfiguration.Name)
		}
	}

	if len(failedDeletions) != 0 {
		return errors.Errorf("Found but failed to delete the following OSM ValidatingWebhookConfigurations: %+v", failedDeletions)
	}

	return nil
}

// uninstallSecrets uninstalls osm-related secrets from the cluster.
func (d *uninstallClusterWideResourcesCmd) uninstallSecrets() error {
	secrets := []string{
		d.caBundleSecretName,
		constants.CrdConverterCertificateSecretName,
		constants.MutatingWebhookCertificateSecretName,
		constants.ValidatingWebhookCertificateSecretName,
	}

	var failedDeletions []string
	for _, secret := range secrets {
		err := d.clientset.CoreV1().Secrets(d.meshNamespace).Delete(context.Background(), secret, metav1.DeleteOptions{})

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
		return errors.Errorf("Found but failed to delete the following OSM secrets in namespace %s: %+v", d.meshNamespace, failedDeletions)
	}

	return nil
}
