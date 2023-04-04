// Package config defines the constants that are used by multiple other packages within OSM.
package config

const (

	// CNISock defines the sock file
	CNISock = "/var/run/osm-cni.sock"

	// CNICreatePodURL is the route for cni plugin for creating pod
	CNICreatePodURL = "/v1/cni/create-pod"
	// CNIDeletePodURL is the route for cni plugin for deleting pod
	CNIDeletePodURL = "/v1/cni/delete-pod"

	// OsmPodFibEbpfMap is the mount point of osm_pod_fib map
	OsmPodFibEbpfMap = "/sys/fs/bpf/osm_pod_fib"
	// OsmNatFibEbpfMap is the mount point of osm_nat_fib map
	OsmNatFibEbpfMap = "/sys/fs/bpf/osm_nat_fib"
)
