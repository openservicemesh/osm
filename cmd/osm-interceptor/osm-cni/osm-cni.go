// Package main implements osm cni plugin.
package main

import (
	"fmt"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/version"

	"github.com/openservicemesh/osm/pkg/cni/plugin"
	"github.com/openservicemesh/osm/pkg/logger"
)

func init() {
	_ = logger.SetLogLevel("warn")
}

func main() {
	skel.PluginMain(plugin.CmdAdd, plugin.CmdCheck, plugin.CmdDelete, version.All,
		fmt.Sprintf("CNI plugin osm-cni %v", "0.1.0"))
}
