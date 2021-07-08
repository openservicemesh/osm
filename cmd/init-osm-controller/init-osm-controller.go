package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"os"

	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	configClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/version"
)

const (
	meshConfigName          = "osm-mesh-config"
	presetMeshConfigName    = "preset-mesh-config"
	presetMeshConfigJSONKey = "preset-mesh-config.json"
)

var (
	flags = pflag.NewFlagSet("init-osm-controller", pflag.ExitOnError)
	log   = logger.New("init-osm-controller")

	kubeConfigFilePath string
	osmNamespace       string
)

func init() {
	if err := logger.SetLogLevel("info"); err != nil {
		log.Fatal().Err(err).Msg("Error setting log level")
	}

	flags.StringVar(&kubeConfigFilePath, "kubeconfig", "", "Path to Kubernetes config file.")
	flags.StringVar(&osmNamespace, "osm-namespace", "", "Namespace to which OSM belongs to.")
}

func parseFlags() error {
	if err := flags.Parse(os.Args); err != nil {
		return err
	}
	_ = flag.CommandLine.Parse([]string{})
	return nil
}

func validateCLIParams() error {
	if osmNamespace == "" {
		return errors.New("Please specify the OSM namespace using --osm-namespace")
	}
	return nil
}

func createDefaultMeshConfig(presetMeshConfigMap *corev1.ConfigMap) *v1alpha1.MeshConfig {
	presetMeshConfig := presetMeshConfigMap.Data[presetMeshConfigJSONKey]
	presetMeshConfigSpec := v1alpha1.MeshConfigSpec{}
	err := json.Unmarshal([]byte(presetMeshConfig), &presetMeshConfigSpec)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error converting preset-mesh-config json string to meshConfig object")
	}

	return &v1alpha1.MeshConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MeshConfig",
			APIVersion: "config.openservicemesh.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: meshConfigName,
		},
		Spec: presetMeshConfigSpec,
	}
}

func main() {
	log.Info().Msgf("Starting init-osm-controller %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)

	if err := parseFlags(); err != nil {
		log.Fatal().Err(err).Msg("Failed parsing cmd line arguments")
	}

	// This ensures CLI parameters (and dependent values) are correct.
	if err := validateCLIParams(); err != nil {
		log.Fatal().Err(err)
	}

	// Initialize kube config and client
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigFilePath)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed creating kube config (kubeconfig=%s)", kubeConfigFilePath)
	}

	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	configClient, err := configClientset.NewForConfig(kubeConfig)
	if err != nil {
		log.Fatal().Err(err).Msgf("Could not access Kubernetes cluster, check kubeconfig.")
		return
	}

	presetMeshConfigMap, presetConfigErr := kubeClient.CoreV1().ConfigMaps(osmNamespace).Get(context.TODO(), presetMeshConfigName, metav1.GetOptions{})
	_, meshConfigErr := configClient.ConfigV1alpha1().MeshConfigs(osmNamespace).Get(context.TODO(), meshConfigName, metav1.GetOptions{})

	// If the presetMeshConfig could not be loaded and a default meshConfig doesn't exist, return the error
	if presetConfigErr != nil && apierrors.IsNotFound(meshConfigErr) {
		log.Fatal().Err(err).Msgf("Unable to create default meshConfig, as %s could not be found", presetMeshConfigName)
		return
	}

	defaultMeshConfig := createDefaultMeshConfig(presetMeshConfigMap)

	if createdMeshConfig, err := configClient.ConfigV1alpha1().MeshConfigs(osmNamespace).Create(context.TODO(), defaultMeshConfig, metav1.CreateOptions{}); err == nil {
		log.Info().Msgf("MeshConfig created in %s, %v", osmNamespace, createdMeshConfig)
	} else if apierrors.IsAlreadyExists(err) {
		log.Info().Msgf("MeshConfig already exists in %s. Skip creating.", osmNamespace)
	} else {
		log.Fatal().Err(err).Msgf("Error creating default MeshConfig")
	}
}
