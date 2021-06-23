package main

import (
	"fmt"
	"os"

	"github.com/openservicemesh/osm/pkg/utils"
)

func main() {
	fmt.Println("hello!")

	// bookstorePort is the bookstore service's port
	bookstorePort := 14001

	// bookwarehousePort is the bookwarehouse service's port
	bookwarehousePort := 14001

	bookstoreServiceName := utils.GetEnv("BOOKSTORE_SVC", "bookstore")
	bookstoreNamespace := os.Getenv("BOOKSTORE_NAMESPACE")

	warehouseServiceName := "bookwarehouse"
	bookwarehouseNamespace := os.Getenv("BOOKWAREHOUSE_NAMESPACE")



	bookstoreServiceNamespace := fmt.Sprintf("%s.%s", bookstoreServiceName, bookstoreNamespace)
	bookstoreClusterID := os.Getenv("BOOKSTORE_CLUSTER_ID")
	if bookstoreClusterID != "" {
		bookstoreServiceNamespace += fmt.Sprintf(".svc.cluster.%s", bookstoreClusterID)
	}

	bookstoreService := utils.AppendClusterID(fmt.Sprintf("%s:%d", bookstoreServiceNamespace, bookstorePort), bookstoreClusterID) // FQDN

	warehouseService := fmt.Sprintf("%s.%s:%d", warehouseServiceName, bookwarehouseNamespace, bookwarehousePort) // FQDN


	// bookstoreService := fmt.Sprintf("%s.%s:%d", bookstoreServiceName, bookstoreNamespace, bookstorePort) 

	// bookstore-v1.bookstore.svc.cluster.osm-2

	// <bookstore-svc>.<bookstore-namespace>.svc.cluster.<clusterID>:<port>


	

	fmt.Println(bookstoreService)
	fmt.Println(warehouseService)

}
