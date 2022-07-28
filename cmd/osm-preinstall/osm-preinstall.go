package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/version"
)

var log = logger.New("osm-preinstall")

func main() {
	log.Info().Msgf("Starting osm-preinstall %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)

	var verbosity string
	var enforceSingleMesh bool
	var namespace string

	flags := pflag.NewFlagSet("osm-preinstall", pflag.ExitOnError)
	flags.StringVarP(&verbosity, "verbosity", "v", "info", "Set log verbosity level")
	flags.BoolVar(&enforceSingleMesh, "enforce-single-mesh", true, "Enforce only deploying one mesh in the cluster")
	flags.StringVar(&namespace, "namespace", "", "The namespace where the new mesh is to be installed")

	err := flags.Parse(os.Args)
	if err != nil {
		log.Fatal().Err(err).Msg("parsing flags")
	}

	if err := logger.SetLogLevel(verbosity); err != nil {
		log.Fatal().Err(err).Msg("Error setting log level")
	}

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("getting kube client config")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal().Err(err).Msg("creating kube client")
	}

	checks := []func() error{
		singleMeshOK(clientset, enforceSingleMesh),
		namespaceHasNoMesh(clientset, namespace),
	}

	ok := true
	for _, check := range checks {
		if err := check(); err != nil {
			ok = false
			log.Error().Err(err).Msg("check failed")
		}
	}
	if !ok {
		log.Fatal().Msg("checks failed")
	}
	log.Info().Msg("checks OK")
}

func singleMeshOK(clientset kubernetes.Interface, enforceSingleMesh bool) func() error {
	return func() error {
		deps, err := clientset.AppsV1().Deployments("").List(context.Background(), metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(map[string]string{
				constants.AppLabel: constants.OSMControllerName,
			}).String(),
		})
		if err != nil {
			return fmt.Errorf("listing OSM deployments: %w", err)
		}

		var existingMeshes []string
		var existingSingleMeshes []string
		for _, dep := range deps.Items {
			mesh := fmt.Sprintf("namespace: %s, name: %s", dep.Namespace, dep.Labels["meshName"])
			existingMeshes = append(existingMeshes, mesh)
			if dep.Labels["enforceSingleMesh"] == "true" {
				existingSingleMeshes = append(existingSingleMeshes, mesh)
			}
		}

		if len(existingSingleMeshes) > 0 {
			return fmt.Errorf("Mesh(es) %s already enforce it is the only mesh in the cluster, cannot install new meshes", strings.Join(existingSingleMeshes, ", "))
		}

		if enforceSingleMesh && len(existingMeshes) > 0 {
			return fmt.Errorf("Mesh(es) %s already exist so a new mesh enforcing it is the only one cannot be installed", strings.Join(existingMeshes, ", "))
		}

		return nil
	}
}

func namespaceHasNoMesh(clientset kubernetes.Interface, namespace string) func() error {
	return func() error {
		deps, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(map[string]string{
				constants.AppLabel: constants.OSMControllerName,
			}).String(),
		})
		if err != nil {
			return fmt.Errorf("listing osm-controller deployments in namespace %s: %w", namespace, err)
		}
		var meshNames []string
		for _, dep := range deps.Items {
			meshNames = append(meshNames, dep.Labels["meshName"])
		}
		if len(meshNames) > 0 {
			return fmt.Errorf("Namespace %s already contains meshes %v", namespace, meshNames)
		}
		return nil
	}
}
