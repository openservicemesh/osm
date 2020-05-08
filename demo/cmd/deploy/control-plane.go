package main

import (
	"fmt"
	"os"

	"github.com/open-service-mesh/osm/demo/cmd/common"
	"github.com/open-service-mesh/osm/demo/cmd/deploy/metrics"
	"github.com/open-service-mesh/osm/demo/cmd/deploy/osm"
)

func main() {
	namespace := os.Getenv(common.KubeNamespaceEnvVar)
	if namespace == "" {
		fmt.Println("Empty namespace")
		os.Exit(1)
	}
	clientset := common.GetClient()
	err := osm.DeployOSM(clientset, namespace)
	if err != nil {
		fmt.Println("Error creating osm: ", err)
		os.Exit(1)
	}
	err = metrics.DeployPrometheus(clientset, namespace)
	if err != nil {
		fmt.Println("Error creating prometheus: ", err)
		os.Exit(1)
	}
}
