package configurator

import (
	"fmt"
	v1 "github.com/open-service-mesh/osm/pkg/apis/osmconfig/v1"
)

func (c Client) getConfig() *v1.OSMConfig {
	key := fmt.Sprintf("%s/%s", c.configCRDNamespace, c.configCRDName)
	item, exists, err := c.cache.GetByKey(key)
	if err != nil {
		log.Error().Err(err).Msgf("Error fetching OSM Config CRD %s/%s from cache", c.configCRDNamespace, c.configCRDName)
		return nil
	}
	if !exists {
		return nil
	}
	osmConfig, ok := item.(*v1.OSMConfig)
	if !ok {
		log.Error().Msg("Object found in OSM Config CRD cache is not OSM CRD")
		return nil
	}
	return osmConfig
}

// IsMonitoredNamespace returns a boolean indicating if the namespace is among the list of monitored namespaces
func (c Client) IsMonitoredNamespace(namespace string) bool {
	log.Trace().Msgf("Checking whether namespace %s is in the list of observed namespaces", namespace)
	cfg := c.getConfig()
	if cfg == nil {
		return false
	}
	log.Trace().Msgf("OSM Config CRD Spec: %+v", cfg.Spec)
	for _, ns := range cfg.Spec.Namespaces {
		if namespace == ns {
			return true
		}
	}
	return false
}
