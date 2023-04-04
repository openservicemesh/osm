// Package main implements osm interceptor.
package main

import (
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/openservicemesh/osm/pkg/cni/config"
	"github.com/openservicemesh/osm/pkg/cni/controller/cniserver"
	"github.com/openservicemesh/osm/pkg/cni/controller/helpers"
	"github.com/openservicemesh/osm/pkg/cni/controller/podwatcher"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/version"
)

var (
	verbosity         string
	meshName          string // An ID that uniquely identifies an OSM instance
	kubeConfigFile    string
	osmNamespace      string
	osmMeshConfigName string
	osmVersion        string

	scheme = runtime.NewScheme()

	flags = pflag.NewFlagSet(`osm-interceptor`, pflag.ExitOnError)
	log   = logger.New("osm-interceptor/main")
)

func init() {
	flags.StringVarP(&verbosity, "verbosity", "v", "info", "Set log verbosity level")
	flags.StringVar(&meshName, "mesh-name", "", "OSM mesh name")
	flags.StringVar(&kubeConfigFile, "kubeconfig", "", "Path to Kubernetes config file.")
	flags.StringVar(&osmNamespace, "osm-namespace", "", "Namespace to which OSM belongs to.")
	flags.StringVar(&osmMeshConfigName, "osm-config-name", "osm-mesh-config", "Name of the OSM MeshConfig")
	flags.StringVar(&osmVersion, "osm-version", "", "Version of OSM")

	// Get some flags from commands
	flags.BoolVarP(&config.KernelTracing, "kernel-tracing", "d", false, "KernelTracing mode")
	flags.BoolVarP(&config.IsKind, "kind", "k", false, "Enable when Kubernetes is running in Kind")
	flags.BoolVar(&config.EnableCNI, "cni-mode", false, "Enable CNI plugin")
	flags.StringVar(&config.HostProc, "host-proc", "/host/proc", "/proc mount path")
	flags.StringVar(&config.CNIBinDir, "cni-bin-dir", "/host/opt/cni/bin", "/opt/cni/bin mount path")
	flags.StringVar(&config.CNIConfigDir, "cni-config-dir", "/host/etc/cni/net.d", "/etc/cni/net.d mount path")
	flags.StringVar(&config.HostVarRun, "host-var-run", "/host/var/run", "/var/run mount path")

	_ = clientgoscheme.AddToScheme(scheme)
}

func parseFlags() error {
	if err := flags.Parse(os.Args); err != nil {
		return err
	}
	_ = flag.CommandLine.Parse([]string{})
	return nil
}

// validateCLIParams contains all checks necessary that various permutations of the CLI flags are consistent
func validateCLIParams() error {
	if meshName == "" {
		return fmt.Errorf("Please specify the mesh name using --mesh-name")
	}

	if osmNamespace == "" {
		return fmt.Errorf("Please specify the OSM namespace using --osm-namespace")
	}

	return nil
}

func main() {
	log.Info().Msgf("Starting osm-interceptor %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
	if err := parseFlags(); err != nil {
		log.Fatal().Err(err).Msg("Error parsing cmd line arguments")
	}
	if err := logger.SetLogLevel(verbosity); err != nil {
		log.Fatal().Err(err).Msg("Error setting log level")
	}

	// This ensures CLI parameters (and dependent values) are correct.
	if err := validateCLIParams(); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InvalidCLIParameters, "Error validating CLI parameters")
	}

	// Initialize kube config and client
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error creating kube config (kubeconfig=%s)", kubeConfigFile)
	}
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)

	if err = helpers.LoadProgs(config.EnableCNI, config.KernelTracing); err != nil {
		log.Fatal().Msgf("failed to load ebpf programs: %v", err)
	}

	stop := make(chan struct{}, 1)
	if config.EnableCNI {
		cniReady := make(chan struct{}, 1)
		s := cniserver.NewServer(path.Join("/host", config.CNISock), "/sys/fs/bpf", cniReady, stop)
		if err = s.Start(); err != nil {
			log.Fatal().Err(err)
		}
	}
	if err = podwatcher.Run(kubeClient, stop); err != nil {
		log.Fatal().Err(err)
	}

	log.Info().Msgf("Stopping osm-interceptor %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
}
