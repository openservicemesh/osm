package azure

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"

	"github.com/deislabs/smc/pkg/mesh"
	"github.com/deislabs/smc/pkg/utils"
)

func (az *Client) getVM() (map[mesh.ServiceName][]mesh.IP, error) {
	res := make(map[mesh.ServiceName][]mesh.IP)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	list, err := az.vmClient.List(ctx, az.resourceGroup)
	glog.Infof("[EDS][ServiceName Discovery] List VMs for resource group: %s", az.resourceGroup)
	if err != nil {
		glog.Error("Could not retrieve all VMSS: ", err)
	}
	for _, vm := range list.Values() {
		netIf := (*vm.NetworkProfile.NetworkInterfaces)[0].ID
		glog.Infof("[EDS][ServiceName Discovery] Found VM %s with NIC %s", *vm.Name, *netIf)
		ctxA, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		ifName := utils.GetLastChunkOfSlashed(*netIf)
		nif, err := az.netClient.Get(ctxA, az.resourceGroup, ifName, "")
		if err != nil {
			glog.Error("Could got get net interface ", ifName, " for resource group ", az.resourceGroup, err)
			cancel()
			continue
		}

		for _, ipConf := range *nif.IPConfigurations {
			if ipConf.PrivateIPAddress != nil {
				// TODO: gotta shoehorn the Kubernetes namespace here because we form the unique keys for the services
				// from the Namespace and teh ServiceName from the TrafficSplit CRD
				svcID := mesh.ServiceName(fmt.Sprintf("%s/%s", az.namespace, *vm.ID))
				res[svcID] = append(res[svcID], mesh.IP(*ipConf.PrivateIPAddress))
			}
		}
		cancel()
	}
	return res, nil
}
