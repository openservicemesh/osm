package configurator

import (
	"encoding/json"
)

// The functions in this file implement the configurator.Configurator interface

// GetOSMNamespace returns the namespace in which the OSM controller pod resides.
func (c *Client) GetOSMNamespace() string {
	return c.osmNamespace
}

// GetConfigMap returns the ConfigMap in pretty JSON.
func (c *Client) GetConfigMap() ([]byte, error) {
	cm, err := json.MarshalIndent(c.getConfigMap(), "", "    ")
	if err != nil {
		log.Error().Err(err).Msgf("Error marshaling ConfigMap %s: %+v", c.getConfigMapCacheKey(), c.getConfigMap())
		return nil, err
	}
	return cm, nil
}

// IsPermissiveTrafficPolicyMode tells us whether the OSM Control Plane is in permissive mode,
// where all existing traffic is allowed to flow as it is,
// or it is in SMI Spec mode, in which only traffic between source/destinations
// referenced in SMI policies is allowed.
func (c *Client) IsPermissiveTrafficPolicyMode() bool {
	return c.getConfigMap().PermissiveTrafficPolicyMode
}

// IsEgressEnabled determines whether egress is globally enabled in the mesh or not.
func (c *Client) IsEgressEnabled() bool {
	return c.getConfigMap().Egress
}

// GetAnnouncementsChannel returns a channel, which is used to announce when changes have been made to the OSM ConfigMap.
func (c *Client) GetAnnouncementsChannel() <-chan interface{} {
	return c.announcements
}
