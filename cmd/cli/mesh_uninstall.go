package main

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	helmStorage "helm.sh/helm/v3/pkg/storage/driver"
)

const meshUninstallDescription = `
This command will uninstall an instance of the osm control plane
given the mesh name and namespace. It will not delete the namespace
the mesh was installed in.
Only use this in non-production and test environments.
`

type meshUninstallCmd struct {
	out      io.Writer
	in       io.Reader
	meshName string
	force    bool
	client   *action.Uninstall
}

func newMeshUninstall(config *action.Configuration, in io.Reader, out io.Writer) *cobra.Command {
	uninstall := &meshUninstallCmd{
		out: out,
		in:  in,
	}

	cmd := &cobra.Command{
		Use:     "uninstall",
		Aliases: []string{"delete", "del"},
		Short:   "uninstall osm control plane instance",
		Long:    meshUninstallDescription,
		Args:    cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, args []string) error {
			uninstall.client = action.NewUninstall(config)
			return uninstall.run()
		},
	}

	f := cmd.Flags()
	f.StringVar(&uninstall.meshName, "mesh-name", defaultMeshName, "Name of the service mesh")
	f.BoolVarP(&uninstall.force, "force", "f", false, "Attempt to uninstall the osm control plane instance without prompting for confirmation.  If the control plane with specified mesh name does not exist, do not display a diagnostic message or modify the exit status to reflect an error.")

	return cmd
}

func (d *meshUninstallCmd) run() error {
	if !d.force {
		confirm, err := confirm(d.in, d.out, fmt.Sprintf("Uninstall OSM [mesh name: %s] ?", d.meshName), 3)
		if !confirm || err != nil {
			return err
		}
	}

	_, err := d.client.Run(d.meshName)
	if err != nil && errors.Cause(err) == helmStorage.ErrReleaseNotFound {
		if d.force {
			return nil
		}
		return errors.Errorf("No OSM control plane with mesh name [%s] found in namespace [%s]", d.meshName, settings.Namespace())
	}

	if err == nil {
		fmt.Fprintf(d.out, "OSM [mesh name: %s] uninstalled\n", d.meshName)
	}

	return err
}
