package main

import (
	"io"

	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"

	"github.com/openservicemesh/osm/pkg/constants"
)

const metricsDescription = `
This command consists of multiple subcommands related to managing metrics
associated with osm.
`

func newMetricsCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "manage metrics",
		Long:  metricsDescription,
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newMetricsEnable(out))
	cmd.AddCommand(newMetricsDisable(out))

	return cmd
}

// isMonitoredNamespace returns true if the Namespace is correctly annotated for monitoring given a set of existing meshes
func isMonitoredNamespace(ns corev1.Namespace, meshList mapset.Set) (bool, error) {
	// Check if the namespace has the resource monitor annotation
	meshName, monitored := ns.Labels[constants.OSMKubeResourceMonitorAnnotation]
	if !monitored {
		return false, nil
	}
	if meshName == "" {
		return false, errors.Errorf("Label %q on namespace %q cannot be empty",
			constants.OSMKubeResourceMonitorAnnotation, ns.Name)
	}
	if !meshList.Contains(meshName) {
		return false, errors.Errorf("Invalid mesh name %q used with label %q on namespace %q, must be one of %v",
			meshName, constants.OSMKubeResourceMonitorAnnotation, ns.Name, meshList.ToSlice())
	}

	return true, nil
}
