package azure

import (
	"context"
	"net"
	"time"

	"github.com/golang/glog"
	"github.com/open-service-mesh/osm/pkg/log/level"
)

func (az *Client) getVMSS(rg resourceGroup, vmID azureID) ([]net.IP, error) {
	glog.V(level.Trace).Infof("[%s] Fetching IPS of VMSS for %s in resource group: %s", packageName, vmID, rg)
	var ips []net.IP
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	glog.V(level.Debug).Infof("[%s] List all VMSS for resource group: %s", packageName, rg)
	list, err := az.vmssClient.List(ctx, string(rg))
	if err != nil {
		glog.Errorf("[%s] Could not retrieve all VMSS: %s", packageName, err)
	}
	for _, vmss := range list.Values() {
		if *vmss.ID != string(vmID) {
			continue
		}
		glog.Infof("[%s] Found VMSS %s", packageName, *vmss.Name)
		// TODO(draychev): get the IP address of each sub-instance and append to the list of Endpoints
	}
	return ips, nil
}
