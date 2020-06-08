package cli

/*
Package cli describes the operating environment for the OSM cli.
*/

import (
	"os"

	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	defaultOSMNamespace = "osm-system"
	osmNamespaceEnvVar  = "OSM_NAMESPACE"
)

// EnvSettings describes all of the cli environment settings
type EnvSettings struct {
	namespace string
	config    *genericclioptions.ConfigFlags
}

// New relevant environment variables set and returns EnvSettings
func New() *EnvSettings {
	env := &EnvSettings{
		namespace: envOr(osmNamespaceEnvVar, defaultOSMNamespace),
	}

	// bind to kubernetes config flags
	env.config = &genericclioptions.ConfigFlags{
		Namespace: &env.namespace,
	}
	return env
}

func envOr(name, defaultVal string) string {
	if v, ok := os.LookupEnv(name); ok {
		return v
	}
	return defaultVal
}

// AddFlags binds flags to the given flagset.
func (s *EnvSettings) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&s.namespace, "namespace", "n", s.namespace, "namespace scope for this request")
}

// EnvVars returns a map of all OSM related environment variables
func (s *EnvSettings) EnvVars() map[string]string {
	return map[string]string{
		osmNamespaceEnvVar: s.Namespace(),
	}
}

// RESTClientGetter gets the kubeconfig from EnvSettings
func (s *EnvSettings) RESTClientGetter() genericclioptions.RESTClientGetter {
	return s.config
}

// Namespace gets the namespace from the configuration
func (s *EnvSettings) Namespace() string {
	if ns, _, err := s.config.ToRawKubeConfigLoader().Namespace(); err == nil {
		return ns
	}
	return "default"
}
