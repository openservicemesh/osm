package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"

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
	clientset     kubernetes.Interface
	remoteVersion remoteVersionGetter
}

type remoteVersionGetter interface {
	proxyGetMeshVersion(pod string, namespace string, clientset kubernetes.Interface) (*version.Info, error)
}

type remoteVersion struct{}

type remoteVersionInfo struct {
	meshName string
	version  *version.Info
}

type versionInfo struct {
	cliVersionInfo    *version.Info
	remoteVersionInfo *remoteVersionInfo
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
			var versionInfo versionInfo
			var err error

			cliVersionInfo := version.GetInfo()
			versionInfo.cliVersionInfo = &cliVersionInfo

			if !versionCmd.clientOnly {
				err = versionCmd.setKubeClientset()
				if err == nil {
					versionCmd.namespace = settings.Namespace()
					versionCmd.remoteVersion = &remoteVersion{}
					versionInfo.remoteVersionInfo, err = versionCmd.getMeshVersion()
				}
			}

			versionCmd.outputVersionInfo(versionInfo)
			if err != nil {
				return errors.Wrap(err, "Failed to get mesh version")
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVar(&versionCmd.clientOnly, "client-only", false, "only show the OSM CLI version")

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
		meshName: controllerPod.Labels[constants.OSMAppInstanceLabelKey],
		version:  version,
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

func (v *versionCmd) outputVersionInfo(versionInfo versionInfo) {
	fmt.Fprintf(v.out, "CLI Version: %#v\n", *versionInfo.cliVersionInfo)
	if versionInfo.remoteVersionInfo != nil {
		if versionInfo.remoteVersionInfo.meshName != "" {
			fmt.Fprintf(v.out, "Mesh [%s] Version: %#v\n", versionInfo.remoteVersionInfo.meshName, *versionInfo.remoteVersionInfo.version)
		} else {
			fmt.Fprintf(v.out, "Mesh Version: No control plane found in namespace [%s]\n", v.namespace)
		}
	}
}
