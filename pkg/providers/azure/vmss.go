package azure

import (
	"context"
	"time"

	"github.com/deislabs/smc/pkg/mesh"
	"github.com/golang/glog"
)

func (az *Client) getVMSS(rg resourceGroup, vmID mesh.AzureID) ([]mesh.IP, error) {
	glog.V(7).Infof("[azure] Fetching IPS of VMSS for %s in resource group: %s", vmID, rg)
	var ips []mesh.IP
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	glog.Info("[azure] List all VMSS for resource group: ", rg)
	list, err := az.vmssClient.List(ctx, string(rg))
	if err != nil {
		glog.Error("[azure] Could not retrieve all VMSS: ", err)
	}
	for _, vmss := range list.Values() {
		if *vmss.ID != string(vmID) {
			continue
		}
		glog.Infof("[azure] Found VMSS %s", *vmss.Name)
		// TODO(draychev): get the IP address of each sub-instance and append to the list of IPs
	}
	return ips, nil
}
