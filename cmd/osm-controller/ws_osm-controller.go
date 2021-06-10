package main

import (
	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/endpoint/providers/remote"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
	"github.com/openservicemesh/osm/pkg/smi"
	"github.com/openservicemesh/osm/pkg/witesand"
	clientset "k8s.io/client-go/kubernetes"
)

var (
	enableRemoteCluster bool
	clusterId           string
	osmControllerName   string
	remoteProvider      *remote.Client
	witesandCatalog     *witesand.WitesandCatalog
	m                   *catalog.MeshCatalog
)

func wsinit() {
	flags.BoolVar(&enableRemoteCluster, "enable-remote-cluster", false, "Enable Remote cluster")
	flags.StringVar(&clusterId, "cluster-id", "master", "Cluster Id")
	flags.StringVar(&osmControllerName, "osm-controller-name", "osm-controller", "Service name of osm-controller.")
}


func wsRemoteCluster(kubeClient *clientset.Clientset, err error, stop chan struct{}, meshSpec smi.MeshSpec, endpointsProviders []endpoint.Provider) (error, []endpoint.Provider) {
	log.Info().Msgf("enableRemoteCluster:%t clusterId:%s", enableRemoteCluster, clusterId)
	witesandCatalog = witesand.NewWitesandCatalog(kubeClient, clusterId)
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating Witesand catalog")
	}
	if enableRemoteCluster {
		remoteProvider, err = remote.NewProvider(kubeClient, witesandCatalog, clusterId, stop, meshSpec, constants.RemoteProviderName)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to initialize remote provider")
		}
		endpointsProviders = append(endpointsProviders, remoteProvider)
	}
	return err, endpointsProviders
}

