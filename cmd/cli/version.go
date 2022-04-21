package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/constants"
	httpserverconstants "github.com/openservicemesh/osm/pkg/httpserver/constants"
	"github.com/openservicemesh/osm/pkg/version"
)

const versionHelp = `
This command prints the OSM CLI and remote mesh version information
`

type versionCmd struct {
	out           io.Writer
	namespace     string
	clientOnly    bool
	versionOnly   bool
	config        *rest.Config
	clientset     kubernetes.Interface
	remoteVersion remoteVersionGetter
}

type remoteVersionGetter interface {
	proxyGetMeshVersion(pod string, namespace string, clientset kubernetes.Interface) (*version.Info, error)
}

type remoteVersion struct{}

type remoteVersionInfo struct {
	meshName  string
	namespace string
	version   *version.Info
}

type versionInfo struct {
	cliVersionInfo        *version.Info
	remoteVersionInfoList []*remoteVersionInfo
}

func newVersionCmd(out io.Writer) *cobra.Command {
	versionCmd := &versionCmd{
		out: out,
	}
	cmd := &cobra.Command{
		Use:   "version",
		Short: "osm cli and mesh version",
		Long:  versionHelp,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var verInfo versionInfo
			var multiError *multierror.Error

			versionCmd.remoteVersion = &remoteVersion{}

			cliVersionInfo := version.GetInfo()
			verInfo.cliVersionInfo = &cliVersionInfo
			fmt.Fprintf(versionCmd.out, "CLI Version: %#v\n", *verInfo.cliVersionInfo)
			if versionCmd.clientOnly {
				return nil
			}

			if err := versionCmd.setKubeClientset(); err != nil {
				return err
			}

			meshInfoList, err := getMeshInfoList(versionCmd.config, versionCmd.clientset)
			if err != nil {
				return errors.Wrapf(err, "unable to list meshes within the cluster")
			}

			for _, m := range meshInfoList {
				versionCmd.namespace = m.namespace
				meshVer, err := versionCmd.getMeshVersion()
				if err != nil {
					multiError = multierror.Append(multiError, errors.Wrap(err, fmt.Sprintf("Failed to get mesh version for mesh %s in namespace %s", m.name, m.namespace)))
				}
				verInfo.remoteVersionInfoList = append(verInfo.remoteVersionInfoList, meshVer)
			}

			w := newTabWriter(versionCmd.out)
			fmt.Fprint(w, versionCmd.outputPrettyVersionInfo(verInfo.remoteVersionInfoList))
			_ = w.Flush()

			if !settings.IsManaged() && !versionCmd.versionOnly {
				latestReleaseVersion, err := getLatestReleaseVersion()
				if err != nil {
					multiError = multierror.Append(multiError, errors.Wrapf(err, "Failed to get latest release information"))
				} else if err := outputLatestReleaseVersion(versionCmd.out, latestReleaseVersion, cliVersionInfo.Version); err != nil {
					multiError = multierror.Append(multiError, errors.Wrapf(err, "Failed to output latest release information"))
				}
			}

			return multiError.ErrorOrNil()
		},
	}

	f := cmd.Flags()
	f.BoolVar(&versionCmd.clientOnly, "client-only", false, "only show the OSM CLI version")
	f.BoolVar(&versionCmd.versionOnly, "version-only", false, "only show the OSM version information. Hide warnings and upgrade notifications")

	return cmd
}

func (v *versionCmd) setKubeClientset() error {
	config, err := settings.RESTClientGetter().ToRESTConfig()
	if err != nil {
		return errors.Wrap(err, "Error fetching kubeconfig")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Could not access Kubernetes cluster, check kubeconfig")
	}
	v.clientset = clientset
	return nil
}

func (v *versionCmd) getMeshVersion() (*remoteVersionInfo, error) {
	var version *version.Info
	var versionInfo *remoteVersionInfo

	controllerPods, err := getControllerPods(v.clientset, v.namespace)
	if err != nil {
		return nil, err
	}
	if len(controllerPods.Items) == 0 {
		return &remoteVersionInfo{}, nil
	}

	controllerPod := controllerPods.Items[0]
	version, err = v.remoteVersion.proxyGetMeshVersion(controllerPod.Name, v.namespace, v.clientset)
	if err != nil {
		return nil, err
	}
	versionInfo = &remoteVersionInfo{
		meshName:  controllerPod.Labels[constants.OSMAppInstanceLabelKey],
		namespace: v.namespace,
		version:   version,
	}
	return versionInfo, nil
}

func (r *remoteVersion) proxyGetMeshVersion(pod string, namespace string, clientset kubernetes.Interface) (*version.Info, error) {
	resp, err := clientset.CoreV1().Pods(namespace).ProxyGet("", pod, strconv.Itoa(constants.OSMHTTPServerPort), httpserverconstants.VersionPath, nil).DoRaw(context.TODO())
	if err != nil {
		return nil, errors.Wrapf(err, "Error retrieving mesh version from pod [%s] in namespace [%s]", pod, namespace)
	}
	if len(resp) == 0 {
		return nil, errors.Errorf("Empty response received from pod [%s] in namespace [%s]", pod, namespace)
	}

	versionInfo := &version.Info{}
	err = json.Unmarshal(resp, versionInfo)
	if err != nil {
		return nil, errors.Wrapf(err, "Error unmarshalling retrieved mesh version from pod [%s] in namespace [%s]", pod, namespace)
	}

	return versionInfo, nil
}

func (v *versionCmd) outputPrettyVersionInfo(remoteVerList []*remoteVersionInfo) string {
	table := "\nMESH NAME\tMESH NAMESPACE\tVERSION\tGIT COMMIT\tBUILD DATE\n"
	for _, remoteVersionInfo := range remoteVerList {
		if remoteVersionInfo != nil && remoteVersionInfo.meshName != "" {
			table += fmt.Sprintf(
				"%s\t%s\t%s\t%s\t%s\n",
				remoteVersionInfo.meshName,
				remoteVersionInfo.namespace,
				remoteVersionInfo.version.Version,
				remoteVersionInfo.version.GitCommit,
				remoteVersionInfo.version.BuildDate,
			)
		}
	}
	return table
}
