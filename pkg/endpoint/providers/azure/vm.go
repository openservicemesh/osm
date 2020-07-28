package azure

import (
	"context"
	"net"
	"time"

	"github.com/openservicemesh/osm/pkg/utils"
)

func (az *Client) getVM(rg resourceGroup, vmID azureID) ([]net.IP, error) {
	log.Trace().Msgf("Fetching IPS of VM for %s in resource group: %s", vmID, rg)
	var ips []net.IP
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	list, err := az.vmClient.List(ctx, string(rg))
	log.Info().Msgf("Listing VMs for resource group: %s", rg)
	if err != nil {
		log.Error().Err(err).Msg("Could not retrieve all VMs")
	}
	for _, vm := range list.Values() {
		if *vm.ID != string(vmID) {
			continue
		}
		networkInterfaceID := (*vm.NetworkProfile.NetworkInterfaces)[0].ID
		log.Info().Msgf("Found VM %s with NIC %s", *vm.Name, *networkInterfaceID)
		ctxA, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		interfaceName := utils.GetLastChunkOfSlashed(*networkInterfaceID)
		networkInterfaceName, err := az.netClient.Get(ctxA, string(rg), interfaceName, "")
		if err != nil {
			log.Error().Err(err).Msgf("Could got get network interface %s for resource group %s", interfaceName, rg)
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
