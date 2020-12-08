package scale

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/openservicemesh/osm/tests/framework"
)

// Returns the OSM grafana dashboards of interest to save after the test
func getOSMGrafanaSaveDashboards() []GrafanaPanel {
	return []GrafanaPanel{
		{
			Filename:  "cpu",
			Dashboard: MeshDetails,
			Panel:     CPUPanel,
		},
		{
			Filename:  "mem",
			Dashboard: MeshDetails,
			Panel:     MemRSSPanel,
		},
	}
}

// Returns labels to select OSM controller and OSM-installed Prometheus.
func getOSMTrackResources() []TrackedLabel {
	return []TrackedLabel{
		{
			Namespace: Td.OsmNamespace,
			Label: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": OsmControllerAppLabel,
				},
			},
		},
		{
			Namespace: Td.OsmNamespace,
			Label: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": OsmPrometheusAppLabel,
				},
			},
		},
	}
}
