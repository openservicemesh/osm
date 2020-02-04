package azure

import (
	"context"
	"net"
	"time"

	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/log"
	"github.com/deislabs/smc/pkg/utils"
)

func (az *Client) getVM(rg resourceGroup, vmID azureID) ([]net.IP, error) {
	glog.V(log.LvlTrace).Infof("[azure] Fetching IPS of VM for %s in resource group: %s", vmID, rg)
	var ips []net.IP
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	list, err := az.vmClient.List(ctx, string(rg))
	glog.Infof("[azure] Listing VMs for resource group: %s", rg)
	if err != nil {
		glog.Error("[azure] Could not retrieve all VMs: ", err)
	}
	for _, vm := range list.Values() {
		if *vm.ID != string(vmID) {
			continue
		}
		networkInterfaceID := (*vm.NetworkProfile.NetworkInterfaces)[0].ID
		glog.Infof("[azure] Found VM %s with NIC %s", *vm.Name, *networkInterfaceID)
		ctxA, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		interfaceName := utils.GetLastChunkOfSlashed(*networkInterfaceID)
		networkInterfaceName, err := az.netClient.Get(ctxA, string(rg), interfaceName, "")
		if err != nil {
			glog.Error("[azure] Could got get network interface ", interfaceName, " for resource group ", rg, err)
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
