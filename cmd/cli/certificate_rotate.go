package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"os"
	"time"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/constants"
	configClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	osmConfigClient "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	"github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/scheme"
)

const rotateDesc = `
This command rotates the OSM Root Certificate
`

const meshRotateExample = `
# Rotate the mesh root certificate that is Active in the osm-system namespace
osm alpha certificate rotate -d -y
`

type rotateCmd struct {
	out io.Writer

	meshName        string
	newTrustDomain  string
	clientSet       kubernetes.Interface
	configClient    osmConfigClient.Interface
	prompt          bool
	deleteOld       bool
	waitForRotation time.Duration
	mrcFilePath     string
	mrcName         string
	prompter        func(a ...any) (int, error)
}

func newCertificateRotateCmd(out io.Writer) *cobra.Command {
	rotate := &rotateCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:     "rotate",
		Short:   "rotate the MeshRootCertificate",
		Long:    rotateDesc,
		Example: meshRotateExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			config, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return fmt.Errorf("error fetching kubeconfig: %w", err)
			}

			configClient, err := configClientset.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("could not access Kubernetes cluster, check kubeconfig: %w", err)
			}
			rotate.configClient = configClient

			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("could not access Kubernetes cluster, check kubeconfig: %w", err)
			}
			rotate.clientSet = clientset
			rotate.prompter = fmt.Scanln

			return rotate.run()
		},
	}

	f := cmd.Flags()

	f.StringVar(&rotate.meshName, "mesh-name", defaultMeshName, "Name of the service mesh")
	f.StringVarP(&rotate.newTrustDomain, "trust-domain", "t", "", "If specified the new cert will use this trust domain. Works with certificate provider tresor")
	f.BoolVarP(&rotate.prompt, "accept", "y", false, "when specified it will not prompt for user input")
	f.BoolVarP(&rotate.deleteOld, "delete", "d", false, "when specified it will delete the old MeshRootCertificate, otherwise the MeshRootCertificate will remain as inactive")
	f.DurationVarP(&rotate.waitForRotation, "wait", "w", 60*time.Second, "Time to wait for certificate to propagate to all components at each step")
	f.StringVarP(&rotate.mrcFilePath, "file", "f", "", "File to use for the new MRC. When not supplied the settings from the existing MRC are copied")
	f.StringVarP(&rotate.mrcName, "meshrootcertificate", "c", "", "Existing MeshRootCertificate to rotate to active role using the name in currently configured namespace")
	return cmd
}

func (r *rotateCmd) promptForContinue(msg string) bool {
	if !r.prompt {
		var rotate string
		fmt.Fprintf(r.out, msg+" y/n:")
		_, err := r.prompter(rotate)
		if err != nil {
			fmt.Printf("warning: error reading value from user: %s\n", err)
			return true
		}

		if rotate != "y" {
			return false
		}
	}

	return true
}

func (r *rotateCmd) run() error {
	mrcEnabled, err := r.isMRCEnabled()
	if err != nil {
		return err
	}

	if !mrcEnabled {
		return fmt.Errorf("the MeshConfig must have EnableMeshRootCertificate set to true")
	}

	// TODO(#4891): check cert rotations or metrics instead of using time
	if !r.promptForContinue("Are you sure you want to initiate rotation?") {
		return nil
	}

	oldMrc, err := r.findCurrentInUse()
	if err != nil {
		return err
	}
	fmt.Fprintf(r.out, "Found MeshRootCertificate [%s] for mesh [%s] in namespace [%s]\n\n", oldMrc.Name, r.meshName, settings.Namespace())

	newMrc, err := r.getNewMRCforRotation(oldMrc)
	if err != nil {
		return fmt.Errorf("unable to create or find new MeshRootCertificate: %s", err)
	}
	fmt.Fprintf(r.out, "Using MeshRootCertificate [%s] which is in Inactive role\n", newMrc.Name)

	fmt.Fprintf(r.out, "Moving MeshRootCertificate [%s] to Passive role\n", newMrc.Name)
	if !r.promptForContinue("Are you sure you want to initiate rotation?") {
		return nil
	}
	err = r.updateCertificate(newMrc.Name, v1alpha2.PassiveIntent)
	if err != nil {
		return fmt.Errorf("unable to update MeshRootCertificate [%s] with role [%s]: %s", newMrc.Name, v1alpha2.PassiveIntent, err)
	}
	fmt.Fprintf(r.out, "waiting for %s propagation...\n\n", r.waitForRotation)
	time.Sleep(r.waitForRotation)

	fmt.Fprintf(r.out, "Moving MeshRootCertificate [%s] to Active role\n", newMrc.Name)
	if !r.promptForContinue("Are you sure you want to initiate rotation?") {
		return nil
	}
	err = r.updateCertificate(newMrc.Name, v1alpha2.ActiveIntent)
	if err != nil {
		return fmt.Errorf("unable to update MeshRootCertificate [%s] with role [%s]: %s", newMrc.Name, v1alpha2.ActiveIntent, err)
	}
	fmt.Fprintf(r.out, "waiting for %s propagation...\n\n", r.waitForRotation)
	time.Sleep(r.waitForRotation)

	fmt.Fprintf(r.out, "Moving MeshRootCertificate [%s] to Passive role\n", oldMrc.Name)
	if !r.promptForContinue("Are you sure you want to initiate rotation?") {
		return nil
	}
	err = r.updateCertificate(oldMrc.Name, v1alpha2.PassiveIntent)
	if err != nil {
		return fmt.Errorf("unable to update MeshRootCertificate [%s] with role [%s]: %s", oldMrc.Name, v1alpha2.PassiveIntent, err)
	}
	fmt.Fprintf(r.out, "waiting for %s propagation...\n\n", r.waitForRotation)
	time.Sleep(r.waitForRotation)

	fmt.Fprintf(r.out, "Moving MeshRootCertificate [%s] to Inactive role\n", oldMrc.Name)
	if !r.promptForContinue("Are you sure you to move the old MeshRootCertificate to Inactive?") {
		return nil
	}
	err = r.updateCertificate(oldMrc.Name, v1alpha2.InactiveIntent)
	if err != nil {
		return fmt.Errorf("unable to update MeshRootCertificate [%s] with role [%s]: %s", oldMrc.Name, v1alpha2.InactiveIntent, err)
	}
	fmt.Fprintf(r.out, "waiting for %s propagation...\n\n", r.waitForRotation)
	time.Sleep(r.waitForRotation)

	if r.deleteOld {
		if !r.promptForContinue("Are you sure you want to delete the old MeshRootCertificate?") {
			return nil
		}
		err = r.configClient.ConfigV1alpha2().MeshRootCertificates(settings.Namespace()).Delete(context.Background(), oldMrc.Name, metav1.DeleteOptions{})
		if err != nil {
			fmt.Printf("warning: unable to delete MeshRootCertificate [%s]", oldMrc.Name)
		}

		if oldMrc.Spec.Provider.Tresor != nil {
			err := r.deleteTresorSecret(oldMrc.Spec.Provider.Tresor.CA, newMrc.Spec.Provider.Tresor.CA)
			if err != nil {
				fmt.Printf("warning: unable to delete secret [%s]", oldMrc.Spec.Provider.Tresor.CA.SecretRef.Name)
			}
		}
	}

	fmt.Fprintf(r.out, "\nOSM successfully rotated root certificate for mesh [%s] in namespace [%s]\n", r.meshName, settings.Namespace())
	fmt.Fprintf(r.out, "\nThe new MeshRootCertificate [%s] is now active\n", newMrc.Name)
	return nil
}

func (r *rotateCmd) deleteTresorSecret(oldTresorCA, newTresorCA v1alpha2.TresorCASpec) error {
	if oldTresorCA.SecretRef.Name == newTresorCA.SecretRef.Name &&
		oldTresorCA.SecretRef.Namespace == newTresorCA.SecretRef.Namespace {
		fmt.Printf("warning: unable to delete secret [%s].  tresor's new secret is the same as old secret reference", oldTresorCA.SecretRef.Name)
		return nil
	}

	return r.clientSet.CoreV1().Secrets(oldTresorCA.SecretRef.Namespace).Delete(context.Background(), oldTresorCA.SecretRef.Name, metav1.DeleteOptions{})
}

func (r *rotateCmd) findCurrentInUse() (*v1alpha2.MeshRootCertificate, error) {
	mrcs, err := r.configClient.ConfigV1alpha2().MeshRootCertificates(settings.Namespace()).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("not able to find MeshRootCertificates: %w", err)
	}

	var activeMrc *v1alpha2.MeshRootCertificate
	for _, mrc := range mrcs.Items {
		if mrc.Spec.Intent == v1alpha2.ActiveIntent {
			if activeMrc != nil {
				return nil, fmt.Errorf("can only rotate when there is only one active MRC. Found two: %s and %s", mrc.Name, activeMrc.Name)
			}
			m := mrc
			activeMrc = &m
		}

		if mrc.Spec.Intent == v1alpha2.PassiveIntent {
			return nil, fmt.Errorf("it appears a rotation is in progress. \n\nFound MRC %s in state %s", mrc.Name, mrc.Spec.Intent)
		}
	}

	if activeMrc == nil {
		return nil, fmt.Errorf("not able to find MeshRootCertificate in active state")
	}

	return activeMrc, nil
}

func (r *rotateCmd) updateCertificate(name string, intent v1alpha2.MeshRootCertificateIntent) error {
	mrc, err := r.configClient.ConfigV1alpha2().MeshRootCertificates(settings.Namespace()).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	mrc.Spec.Intent = intent
	_, err = r.configClient.ConfigV1alpha2().MeshRootCertificates(settings.Namespace()).Update(context.Background(), mrc, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (r *rotateCmd) getNewMRCforRotation(mrc *v1alpha2.MeshRootCertificate) (*v1alpha2.MeshRootCertificate, error) {
	// if the user has specified an MRC to use
	if r.mrcName != "" {
		return r.usePreCreatedMRC()
	}

	// if the user has specified a file to use
	if r.mrcFilePath != "" {
		return r.createFromFile()
	}

	// try to create one
	return r.createNewMRC(mrc)
}

func (r *rotateCmd) createFromFile() (*v1alpha2.MeshRootCertificate, error) {
	fmt.Fprintf(r.out, "using file %s\n", r.mrcFilePath)
	decode := scheme.Codecs.UniversalDeserializer().Decode
	file, err := os.ReadFile(r.mrcFilePath)
	if err != nil {
		return nil, err
	}
	obj, _, err := decode([]byte(file), nil, nil)
	if err != nil {
		return nil, err
	}

	newMRC, ok := obj.(*v1alpha2.MeshRootCertificate)
	if !ok {
		return nil, fmt.Errorf("file provided is not recognized as MeshRootCertificate")
	}
	if newMRC.Spec.Intent != v1alpha2.InactiveIntent {
		return nil, fmt.Errorf("file provided must be a MeshRootCertificate in the Inactive role")
	}

	existing, err := r.configClient.ConfigV1alpha2().MeshRootCertificates(settings.Namespace()).Get(context.Background(), newMRC.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to verify that MeshRootCertificate [%s] doesn't already exist. Error: %v", newMRC.Name, err)
	}
	if existing != nil {
		return nil, fmt.Errorf("cannot use MeshRootCertificate [%s] as it already exists in cluster", newMRC.Name)
	}

	return r.configClient.ConfigV1alpha2().MeshRootCertificates(settings.Namespace()).Create(context.Background(), newMRC, metav1.CreateOptions{})
}

func (r *rotateCmd) usePreCreatedMRC() (*v1alpha2.MeshRootCertificate, error) {
	fmt.Fprintf(r.out, "using existing MeshRootCertificate %s\n", r.mrcName)

	existing, err := r.configClient.ConfigV1alpha2().MeshRootCertificates(settings.Namespace()).Get(context.Background(), r.mrcName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to verify that MeshRootCertificate [%s]. Error: %v", existing.Name, err)
	}

	if existing.Spec.Intent != v1alpha2.InactiveIntent {
		return nil, fmt.Errorf("MRC provided must be in Inactive role")
	}

	return existing, nil
}

func (r *rotateCmd) createNewMRC(mrc *v1alpha2.MeshRootCertificate) (*v1alpha2.MeshRootCertificate, error) {
	switch {
	case mrc.Spec.Provider.Tresor != nil:
		trustDomain := mrc.Spec.TrustDomain
		if r.newTrustDomain != "" {
			trustDomain = r.newTrustDomain
		}
		return r.createTresorMRC(trustDomain)
	case mrc.Spec.Provider.Vault != nil:
		// since we can't create the secrets in vault we require them to create a file
		return nil, fmt.Errorf("please use -file or -meshrootcertificate option to provide a MeshRootCertificate Vault provider")
	case mrc.Spec.Provider.CertManager != nil:
		// since we require them to set up cert-manager.io ahead of time we can create it for them
		return nil, fmt.Errorf("please use -file or -meshrootcertificate option to provide a MeshRootCertificate for CertManager provider")
	default:
		return nil, fmt.Errorf("unknown certificate provider: %+v", mrc.Spec.Provider)
	}
}

func generateName() (string, string) {
	var chars = []rune("abcdefghijklmnopqrstuvwxyz1234567890")

	s := make([]rune, 8)
	for i := range s {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		s[i] = chars[num.Int64()]
	}
	return fmt.Sprintf("%s-%x", constants.DefaultMeshRootCertificateName, string(s)), string(s)
}

func (r *rotateCmd) createTresorMRC(trustDomain string) (*v1alpha2.MeshRootCertificate, error) {
	newName, nameSuffix := generateName()
	return r.configClient.ConfigV1alpha2().MeshRootCertificates(settings.Namespace()).Create(
		context.Background(), &v1alpha2.MeshRootCertificate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      newName,
				Namespace: settings.Namespace(),
			},
			Spec: v1alpha2.MeshRootCertificateSpec{
				TrustDomain: trustDomain,
				Intent:      v1alpha2.InactiveIntent,
				Provider: v1alpha2.ProviderSpec{
					Tresor: &v1alpha2.TresorProviderSpec{
						CA: v1alpha2.TresorCASpec{
							SecretRef: v1.SecretReference{
								Name:      fmt.Sprintf("%s-%x", "osm-ca-bundle-", nameSuffix),
								Namespace: settings.Namespace(),
							},
						}},
				},
			},
		}, metav1.CreateOptions{})
}

func (r *rotateCmd) isMRCEnabled() (bool, error) {
	osmNamespace := settings.Namespace()

	meshConfig, err := r.configClient.ConfigV1alpha2().MeshConfigs(osmNamespace).Get(context.TODO(), defaultOsmMeshConfigName, metav1.GetOptions{})

	if err != nil {
		return false, fmt.Errorf("error fetching MeshConfig %s: %w", defaultOsmMeshConfigName, err)
	}
	return meshConfig.Spec.FeatureFlags.EnableMeshRootCertificate, nil
}
