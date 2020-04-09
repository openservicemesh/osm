package main

import (
	"fmt"
	"os"

	"github.com/open-service-mesh/osm/demo/cmd/common"
	"github.com/open-service-mesh/osm/demo/cmd/deploy/metrics"
	"github.com/open-service-mesh/osm/demo/cmd/deploy/xds"
)

func main() {
	namespace := os.Getenv(common.KubeNamespaceEnvVar)
	if namespace == "" {
		fmt.Println("Empty namespace")
		os.Exit(1)
	}
	clientset := common.GetClient()
	err := xds.Xds(clientset, namespace)
	if err != nil {
		fmt.Println("Error creating xds: ", err)
		os.Exit(1)
	}
	err = metrics.Prometheus(clientset, namespace)
	if err != nil {
		fmt.Println("Error creating prometheus: ", err)
		os.Exit(1)
	}
}
