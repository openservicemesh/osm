/*
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

This file is inspired by the way Helm handles environment variables
for the Helm CLI https://github.com/helm/helm/blob/master/cmd/helm/env.go
*/
package main

import (
	"context"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const updateHelp = `
This command updates a deployed osm controlplane configuration on a Kubernetes cluster.
`

const (
	defaultMeshNamespace      = "osm-system"
)

type updateCmd struct {
	out       io.Writer
	meshName  string
	clientSet kubernetes.Interface
}

func newUpdateCmd(config *action.Configuration, out io.Writer) *cobra.Command {
	inst := &updateCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "update osm control plane configuration",
		Long:  updateHelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			envVars := settings.EnvVars()

			kubeconfig, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return errors.Errorf("Error fetching kubeconfig")
			}

			clientset, err := kubernetes.NewForConfig(kubeconfig)
			if err != nil {
				return errors.Errorf("Could not access Kubernetes cluster. Check kubeconfig")
			}

			inst.clientSet = clientset
			return inst.run(envVars)
		},
	}

	f := cmd.Flags()
	f.StringVar(&updt.meshName, "mesh-name", defaultMeshName, "Name of the service mesh")
	return cmd
}

func (u *updateCmd) run(envVars map[string]string) error {
	ns, ok := envVars["K8S_NAMESPACE"]
	if !ok {
		ns = defaultMeshNamespace
	}
	cm, err := u.clientSet.CoreV1().ConfigMaps(ns).Get(context.Background(), u.meshName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	cm.Data["permissive_traffic_policy_mode"] = "true"
	return nil
}
