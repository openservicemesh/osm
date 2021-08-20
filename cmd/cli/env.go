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
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

const envHelp = `
This command prints out all the environment information used by OSM
`

func newEnvCmd(stdout io.Writer, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "osm client environment information",
		Long:  envHelp,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			envConfig := settings.Config()
			config, err := yaml.Marshal(&envConfig)
			if err != nil {
				fmt.Fprintf(stderr, "Error redendering environment information: %s", err)
				return
			}
			fmt.Printf("--- \n%s\n", string(config))
		},
	}
	return cmd
}
