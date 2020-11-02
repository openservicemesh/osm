/*
Package cli describes the operating environment for the OSM cli and
includes convenience functions for the OSM cli.

Copyright The Helm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Copyright 2020 The OSM contributors

Licensed under the MIT License
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

This package is heavily inspired by how the Helm project handles
environment variables: https://github.com/helm/helm/blob/master/pkg/cli/environment.go
*/
package cli

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
	fs.StringVar(&s.namespace, "osm-namespace", s.namespace, "namespace for osm control plane")
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
