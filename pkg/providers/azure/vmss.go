package azure

import (
	"context"
	"time"

	"github.com/deislabs/smc/pkg/mesh"
	"github.com/golang/glog"
)

func (az *Client) getVMSS() (map[mesh.ServiceName][]mesh.IP, error) {
	ips := make(map[mesh.ServiceName][]mesh.IP)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	glog.Info("List all VMSS for resource group: ", az.resourceGroup)
	list, err := az.vmssClient.List(ctx, az.resourceGroup)
	if err != nil {
		glog.Error("Could not retrieve all VMSS: ", err)
	}
	for _, vmss := range list.Values() {
		glog.Infof("Found VMSS %s", *vmss.Name)
		// TODO(draychev): get the IP address of each sub-instance and append to the list of IPs
	}
	return ips, nil
}
