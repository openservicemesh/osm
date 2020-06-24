package configurator

import (
	"encoding/json"
)

// GetOSMNamespace returns the namespace in which the OSM controller pod resides.
func (c *ConfigMapWatcher) GetOSMNamespace() string {
	c.Lock()
	ns := c.osmNamespace
	c.Unlock()
	return ns
}

// GetConfigMap returns the ConfigMap in JSON format.
func (c *ConfigMapWatcher) GetConfigMap() ([]byte, error) {
	c.Lock()
	defer c.Unlock()
	cm, err := json.MarshalIndent(c.configMap, "", "    ")
	if err != nil {
		log.Error().Err(err).Msgf("Error marshaling ConfigMap: %+v", c.configMap)
		return nil, err
	}
	return cm, nil

}
