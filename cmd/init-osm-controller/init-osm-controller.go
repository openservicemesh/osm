package main

import (
	"context"
	"flag"
	"os"

	"github.com/spf13/pflag"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	"github.com/openservicemesh/osm/pkg/cli"
	configClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/version"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

const meshConfigName = "osm-mesh-config"

var settings = cli.New()

var (
	flags = pflag.NewFlagSet("init-osm-controller", pflag.ExitOnError)
	log   = logger.New("init-osm-controller")

	meshName           string // An name of service mesh it tries to initialize
	kubeConfigFilePath string
	osmNamespace       string
)

func init() {
	if err := logger.SetLogLevel("info"); err != nil {
		log.Fatal().Err(err).Msg("Error setting log level")
	}

	flags.StringVar(&meshName, "mesh-name", "", "OSM mesh name")
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

func createDefaultMeshConfig() *v1alpha1.MeshConfig {
	return &v1alpha1.MeshConfig{
		TypeMeta: v1.TypeMeta{
			Kind:       "MeshConfig",
			APIVersion: "config.openservicemesh.io/v1alpha1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name: meshConfigName,
		},
		Spec: v1alpha1.MeshConfigSpec{
			Sidecar: v1alpha1.SidecarSpec{
				LogLevel:                      "error",
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

	meshConfig := createDefaultMeshConfig()

	if _, err := configClient.ConfigV1alpha1().MeshConfigs(settings.Namespace()).Create(context.TODO(), meshConfig, metav1.CreateOptions{}); err == nil {
		log.Info().Msg("MeshConfig created in kubernetes")
	} else if apierrors.IsAlreadyExists(err) {
		log.Info().Msg("MeshConfig already exists in kubernetes. Skip creating.")
	} else {
		log.Fatal().Err(err).Msgf("Error creating default MeshConfig")
	}
}
