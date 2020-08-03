package configurator

import (
	"encoding/json"
	"fmt"
	"github.com/openservicemesh/osm/pkg/constants"
	"net"
	"sort"
	"strings"
)

// The functions in this file implement the configurator.Configurator interface

// GetOSMNamespace returns the namespace in which the OSM controller pod resides.
func (c *Client) GetOSMNamespace() string {
	return c.osmNamespace
}

func marshalConfigToJSON(config *osmConfig) ([]byte, error) {
	return json.MarshalIndent(config, "", "    ")
}

// GetConfigMap returns the ConfigMap in pretty JSON.
func (c *Client) GetConfigMap() ([]byte, error) {
	cm, err := marshalConfigToJSON(c.getConfigMap())
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

// IsPrometheusScrapingEnabled determines whether Prometheus is enabled for scraping metrics
func (c *Client) IsPrometheusScrapingEnabled() bool {
	return c.getConfigMap().PrometheusScraping
}

// IsZipkinTracingEnabled determines whether Zipkin tracing is enabled
func (c *Client) IsZipkinTracingEnabled() bool {
	return c.getConfigMap().ZipkinTracing
}

// GetZipkinHost is the host to which we send Zipkin spans
func (c *Client) GetZipkinHost() string {
	zipkinAddress := c.getConfigMap().ZipkinAddress
	if zipkinAddress != "" {
		return zipkinAddress
	}
	return fmt.Sprintf("%s.%s.svc.cluster.local", constants.DefaultZipkinAddress, c.GetOSMNamespace())
}

// GetZipkinPort returns the Zipkin port
func (c *Client) GetZipkinPort() uint32 {
	zipkinPort := c.getConfigMap().ZipkinPort
	if zipkinPort != 0 {
		return uint32(zipkinPort)
	}
	return constants.DefaultZipkinPort
}

// GetZipkinEndpoint returns the Zipkin endpoint
func (c *Client) GetZipkinEndpoint() string {
	zipkinEndpoint := c.getConfigMap().ZipkinEndpoint
	if zipkinEndpoint != "" {
		return zipkinEndpoint
	}
	return constants.DefaultZipkinEndpoint
}

// GetMeshCIDRRanges returns a list of mesh CIDR ranges
func (c *Client) GetMeshCIDRRanges() []string {
	noSpaces := strings.ReplaceAll(c.getConfigMap().MeshCIDRRanges, " ", ",")
	commaSeparatedCIDRs := strings.Split(noSpaces, ",")

	cidrSet := make(map[string]interface{})
	for _, cidr := range commaSeparatedCIDRs {
		trimmedCIDR := strings.Trim(cidr, " ")
		if len(trimmedCIDR) == 0 {
			continue
		}

		_, _, err := net.ParseCIDR(trimmedCIDR)
		if err != nil {
			log.Error().Err(err).Msgf("Found incorrectly formatted in-mesh CIDR %s from ConfigMap %s/%s; Skipping CIDR", trimmedCIDR, c.osmNamespace, c.osmConfigMapName)
			continue
		}

		cidrSet[trimmedCIDR] = nil
	}

	var cidrs []string
	for cidr := range cidrSet {
		cidrs = append(cidrs, cidr)
	}

	sort.Strings(cidrs)

	return cidrs
}

// UseHTTPSIngress determines whether traffic between ingress and backend pods should use HTTPS protocol
func (c *Client) UseHTTPSIngress() bool {
	return c.getConfigMap().UseHTTPSIngress
}

// GetAnnouncementsChannel returns a channel, which is used to announce when changes have been made to the OSM ConfigMap.
func (c *Client) GetAnnouncementsChannel() <-chan interface{} {
	return c.announcements
}
