package helpers

import (
	"fmt"

	"github.com/cilium/ebpf"

	"github.com/openservicemesh/osm/pkg/cni/config"
)

var (
	podFibMap *ebpf.Map
	natFibMap *ebpf.Map
)

// InitLoadPinnedMap init, load and pinned mapsß
func InitLoadPinnedMap() error {
	var err error
	podFibMap, err = ebpf.LoadPinnedMap(config.OsmPodFibEbpfMap, &ebpf.LoadPinOptions{})
	if err != nil {
		return fmt.Errorf("load map[%s] error: %v", config.OsmPodFibEbpfMap, err)
	}
	natFibMap, err = ebpf.LoadPinnedMap(config.OsmNatFibEbpfMap, &ebpf.LoadPinOptions{})
	if err != nil {
		return fmt.Errorf("load map[%s] error: %v", err, config.OsmNatFibEbpfMap)
	}
	return nil
}

// GetPodFibMap returns pod fib map
func GetPodFibMap() *ebpf.Map {
	if podFibMap == nil {
		_ = InitLoadPinnedMap()
	}
	return podFibMap
}

// GetNatFibMap returns nat fib map
func GetNatFibMap() *ebpf.Map {
	if natFibMap == nil {
		_ = InitLoadPinnedMap()
	}
	return natFibMap
}
