package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/version"
)

const versionHelp = `
This command prints out the following version information:
(1) the OSM version installed in the mesh name and namespace,
(2) the local OSM CLI version,
(3) the latest OSM release version available.

The mesh to check for version information is identified
by its mesh name and namespace.
Use the --mesh-name and --osm-namespace flags to specify the mesh to
check for version information.
`

const versionExample = `
# Print the osm version in the mesh "osm1" and namespace "osm-ns1".
# Also print the local CLI version and the latest release version available.
osm version --mesh-name osm1 --osm-namespace osm-ns1
`

type versionCmd struct {
	meshName  string
	remote    bool
	out       io.Writer
	config    *rest.Config
	clientSet kubernetes.Interface
}

const (
	osmReleaseRepoOwner = "openservicemesh"
	osmReleaseRepoName  = "osm"
)

func newVersionCmd(out io.Writer) *cobra.Command {
	v := &versionCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:     "version",
		Short:   "print 1) version installed in mesh, 2) local cli version, 3) latest available version",
		Long:    versionHelp,
		Example: versionExample,
		Args:    cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, args []string) error {
			config, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return errors.Errorf("Error fetching kubeconfig: %s", err)
			}
			v.config = config
			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				return errors.Errorf("Could not access Kubernetes cluster, check kubeconfig: %s", err)
			}
			v.clientSet = clientset
			return v.run()
		},
	}

	f := cmd.Flags()
	f.BoolVar(&v.remote, "remote", true, "Use remote=false to skip checking osm version installed in the remote mesh.")
	f.StringVar(&v.meshName, "mesh-name", defaultMeshName, "Name of the mesh to check for version information")

	return cmd
}

func (c *versionCmd) run() error {
	// only check remote mesh version information if remote flag is true
	if c.remote {
		meshNamespace := settings.Namespace()
		osmControllerDeployment, err := getMeshControllerDeployment(c.clientSet, c.meshName, meshNamespace)

		if err != nil {
			_, _ = fmt.Fprint(c.out, err.Error()+"\n")
		} else {
			// get mesh version currently installed
			meshVersion := osmControllerDeployment.ObjectMeta.Labels[constants.OSMAppVersionLabelKey]
			if meshVersion == "" {
				meshVersion = "Unknown"
			}
			_, _ = fmt.Fprintf(c.out, "Mesh Name: %s; Mesh Namespace: %s; Mesh Version: %s\n", c.meshName, meshNamespace, meshVersion)
		}
	}

	// print local cli version
	PrintLocalCliVersion(c.out)

	// get latest available release version from the osm repo release API endpoint
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", osmReleaseRepoOwner, osmReleaseRepoName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		_, _ = fmt.Fprintf(c.out, "Unable to fetch latest release version from %s\n", url)
		return nil
	}

	req.Header.Add("Accept", "application/vnd.github.v3+json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		_, _ = fmt.Fprintf(c.out, "Unable to fetch latest release version from %s\n", url)
		return nil
	}
	defer resp.Body.Close() //nolint: errcheck,gosec

	latestReleaseVersionInfo := map[string]interface{}{}
	if err := json.NewDecoder(resp.Body).Decode(&latestReleaseVersionInfo); err != nil {
		_, _ = fmt.Fprintf(c.out, "Unable to decode latest release version information from %s\n", url)
		return nil
	}

	if latestVersion := latestReleaseVersionInfo["tag_name"]; latestVersion != "" {
		_, _ = fmt.Fprintf(c.out, "Latest Available Version: %s\n", latestVersion)
	} else {
		_, _ = fmt.Fprintf(c.out, "Unable to decode latest release version information from %s\n", url)
	}

	return nil
}

// PrintLocalCliVersion prints the version of the CLI
func PrintLocalCliVersion(out io.Writer) {
	_, _ = fmt.Fprintf(out, "Local CLI Version: %s; Commit: %s; Date: %s\n", version.Version, version.GitCommit, version.BuildDate)
}
