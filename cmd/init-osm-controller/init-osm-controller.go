package main

import (
	"context"
	"errors"
	"flag"
	"os"

	"github.com/spf13/pflag"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	configClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/version"

	"k8s.io/client-go/tools/clientcmd"
)

const meshConfigName = "osm-mesh-config"
const presetMeshConfigName = "preset-mesh-config"

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

func createDefaultMeshConfig() *v1alpha1.MeshConfig {
	return &v1alpha1.MeshConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MeshConfig",
			APIVersion: "config.openservicemesh.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: meshConfigName,
		},
		Spec: v1alpha1.MeshConfigSpec{
			Sidecar: v1alpha1.SidecarSpec{
				LogLevel:                      "error",
				EnvoyImage:                    "envoyproxy/envoy-alpine:v1.17.2",
				InitContainerImage:            "openservicemesh/init:v0.8.3",
				EnablePrivilegedInitContainer: false,
				MaxDataPlaneConnections:       0,
			},
			Traffic: v1alpha1.TrafficSpec{
				EnableEgress:                      false,
				UseHTTPSIngress:                   false,
				EnablePermissiveTrafficPolicyMode: false,
			},
			Observability: v1alpha1.ObservabilitySpec{
				EnableDebugServer:  false,
				PrometheusScraping: true,
				Tracing: v1alpha1.TracingSpec{
					Enable: false,
				},
			},
			Certificate: v1alpha1.CertificateSpec{
				ServiceCertValidityDuration: "24h",
			},
		},
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

	configClient, err := configClientset.NewForConfig(kubeConfig)
	if err != nil {
		log.Fatal().Err(err).Msgf("Could not access Kubernetes cluster, check kubeconfig.")
		return
	}

	presetMeshConfig, presetMissing := configClient.ConfigV1alpha1().MeshConfigs(osmNamespace).Get(context.TODO(), presetMeshConfigName, metav1.GetOptions{})

	if _, err := configClient.ConfigV1alpha1().MeshConfigs(osmNamespace).Get(context.TODO(), meshConfigName, metav1.GetOptions{}); err != nil {
		if presetMissing != nil {
			log.Fatal().Err(err).Msg("Error preset meshconfig is missing during OSM installation.")
		}

		meshConfig := &v1alpha1.MeshConfig{
			TypeMeta: metav1.TypeMeta{
				Kind:       "MeshConfig",
				APIVersion: "config.openservicemesh.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: meshConfigName,
			},
			Spec: presetMeshConfig.Spec,
		}

		if createdMeshConfig, err := configClient.ConfigV1alpha1().MeshConfigs(osmNamespace).Create(context.TODO(), meshConfig, metav1.CreateOptions{}); err == nil {
			log.Info().Msgf("MeshConfig created in %s, %v", osmNamespace, createdMeshConfig)
		} else if apierrors.IsAlreadyExists(err) {
			log.Info().Msgf("MeshConfig already exists in %s. Skip creating.", osmNamespace)
		} else {
			log.Fatal().Err(err).Msgf("Error creating default MeshConfig")
		}
	}

	// ensure preset is deleted after initialization
	if err := configClient.ConfigV1alpha1().MeshConfigs(osmNamespace).Delete(context.TODO(), presetMeshConfigName, metav1.DeleteOptions{}); err != nil {
		log.Warn().Msgf("error deleting %s MeshConfig, %s", presetMeshConfigName, err)
	}
}
