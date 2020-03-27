package azure

import (
	"context"
	"net"
	"time"

	"github.com/golang/glog"

	"github.com/open-service-mesh/osm/pkg/log/level"
	"github.com/open-service-mesh/osm/pkg/utils"
)

func (az *Client) getVM(rg resourceGroup, vmID azureID) ([]net.IP, error) {
	glog.V(level.Trace).Infof("[%s] Fetching IPS of VM for %s in resource group: %s", packageName, vmID, rg)
	var ips []net.IP
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	list, err := az.vmClient.List(ctx, string(rg))
	glog.Infof("[%s] Listing VMs for resource group: %s", packageName, rg)
	if err != nil {
		glog.Errorf("[%s] Could not retrieve all VMs: %s", packageName, err)
	}
	for _, vm := range list.Values() {
		if *vm.ID != string(vmID) {
			continue
		}
		networkInterfaceID := (*vm.NetworkProfile.NetworkInterfaces)[0].ID
		glog.Infof("[%s] Found VM %s with NIC %s", packageName, *vm.Name, *networkInterfaceID)
		ctxA, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		interfaceName := utils.GetLastChunkOfSlashed(*networkInterfaceID)
		networkInterfaceName, err := az.netClient.Get(ctxA, string(rg), interfaceName, "")
		if err != nil {
			glog.Errorf("[%s] Could got get network interface %s for resource group %s: %s", packageName, interfaceName, rg, err)
			cancel()
			continue
		}

		for _, ipConf := range *networkInterfaceName.IPConfigurations {
			if ipConf.PrivateIPAddress != nil {
				ips = append(ips, net.IP(*ipConf.PrivateIPAddress))
			}
		}
		cancel()
	}
	return ips, nil
}
