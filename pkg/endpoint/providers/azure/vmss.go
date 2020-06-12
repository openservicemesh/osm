package azure

import (
	"context"
	"net"
	"time"
)

func (az *Client) getVMSS(rg resourceGroup, vmID azureID) ([]net.IP, error) {
	log.Trace().Msgf("Fetching IPS of VMSS for %s in resource group: %s", vmID, rg)
	var ips []net.IP
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	log.Debug().Msgf("List all VMSS for resource group: %s", rg)
	list, err := az.vmssClient.List(ctx, string(rg))
	if err != nil {
		log.Error().Err(err).Msg("Could not retrieve all VMSS")
	}
	for _, vmss := range list.Values() {
		if *vmss.ID != string(vmID) {
			continue
		}
		log.Info().Msgf("Found VMSS %s", *vmss.Name)
		// TODO(draychev): get the IP address of each sub-instance and append to the list of Endpoints
	}
	return ips, nil
}
