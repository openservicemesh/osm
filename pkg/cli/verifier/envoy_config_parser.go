package verifier

import (
	adminv3 "github.com/envoyproxy/go-control-plane/envoy/admin/v3"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	// All of these are required for JSON to ConfigDump parsing to work
	_ "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/stream/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/ext_authz/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/health_check/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/lua/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/rbac/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/wasm/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/rbac/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"

	"github.com/openservicemesh/osm/pkg/cli"
	"github.com/openservicemesh/osm/pkg/constants"
)

// ConfigGetter is an interface for getting Envoy config from Pods' sidecars
type ConfigGetter interface {
	// Get returns the Envoy config
	Get() (*Config, error)
}

// Config is Envoy config dump.
type Config struct {
	// Boostrap is an Envoy xDS proto.
	Boostrap adminv3.BootstrapConfigDump

	// Clusters is an Envoy xDS proto.
	Clusters adminv3.ClustersConfigDump

	// Endpoints is an Envoy xDS proto.
	Endpoints adminv3.EndpointsConfigDump

	// Listeners is an Envoy xDS proto.
	Listeners adminv3.ListenersConfigDump

	// SecretsConfigDump is an Envoy xDS proto.
	SecretsConfigDump adminv3.SecretsConfigDump

	// ScopedRoutesConfigDump is an Envoy xDS proto.
	ScopedRoutesConfigDump adminv3.ScopedRoutesConfigDump

	// Routes is an Envoy xDS proto.
	Routes adminv3.RoutesConfigDump
}

// PodConfigGetter implements the ConfigGetter interface
type PodConfigGetter struct {
	restConfig *rest.Config
	kubeClient kubernetes.Interface
	pod        types.NamespacedName
}

// Get returns the parsed Envoy config dump
func (g PodConfigGetter) Get() (*Config, error) {
	query := "config_dump?include_eds"
	configBytes, err := cli.GetEnvoyProxyConfig(g.kubeClient, g.restConfig, g.pod.Namespace, g.pod.Name, constants.EnvoyAdminPort, query)
	if err != nil {
		return nil, err
	}

	return parseEnvoyConfig(configBytes)
}

// parseEnvoyConfig a Config object representing the parsed config dump
func parseEnvoyConfig(jsonBytes []byte) (*Config, error) {
	var configDump adminv3.ConfigDump
	unmarshal := &protojson.UnmarshalOptions{
		AllowPartial:   true,
		DiscardUnknown: true,
	}
	if err := unmarshal.Unmarshal(jsonBytes, &configDump); err != nil {
		return nil, errors.Errorf("config parse error: %s", err)
	}

	var cfg Config

	for idx, config := range configDump.Configs {
		switch config.TypeUrl {
		case "type.googleapis.com/envoy.admin.v3.BootstrapConfigDump":
			if err := configDump.Configs[idx].UnmarshalTo(&cfg.Boostrap); err != nil {
				return nil, errors.Errorf("error parsing Bootstrap: %s", err)
			}

		case "type.googleapis.com/envoy.admin.v3.ClustersConfigDump":
			if err := configDump.Configs[idx].UnmarshalTo(&cfg.Clusters); err != nil {
				return nil, errors.Errorf("error parsing Clusters: %s", err)
			}

		case "type.googleapis.com/envoy.admin.v3.EndpointsConfigDump":
			if err := configDump.Configs[idx].UnmarshalTo(&cfg.Endpoints); err != nil {
				return nil, errors.Errorf("error parsing Endpoints: %s", err)
			}

		case "type.googleapis.com/envoy.admin.v3.ListenersConfigDump":
			if err := configDump.Configs[idx].UnmarshalTo(&cfg.Listeners); err != nil {
				return nil, errors.Errorf("error parsing Listeners: %s", err)
			}

		case "type.googleapis.com/envoy.admin.v3.RoutesConfigDump":
			if err := configDump.Configs[idx].UnmarshalTo(&cfg.Routes); err != nil {
				return nil, errors.Errorf("error parsing Listeners: %s", err)
			}
		case "type.googleapis.com/envoy.admin.v3.ScopedRoutesConfigDump":
			if err := configDump.Configs[idx].UnmarshalTo(&cfg.ScopedRoutesConfigDump); err != nil {
				return nil, errors.Errorf("error parsing ScopedRoutesConfigDump: %s", err)
			}

		case "type.googleapis.com/envoy.admin.v3.SecretsConfigDump":
			if err := configDump.Configs[idx].UnmarshalTo(&cfg.SecretsConfigDump); err != nil {
				return nil, errors.Errorf("error parsing SecretsConfigDump: %s", err)
			}

		default:
			return nil, errors.Errorf("unrecognized TypeUrl %s", config.TypeUrl)
		}
	}

	return &cfg, nil
}
