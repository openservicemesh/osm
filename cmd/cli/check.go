package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/openservicemesh/osm/pkg/check"
)

const checkDescription = `
This command checks that osm is installed and running properly.

Pre-install validation can be performed by passing the --pre-install flag. When
enabled, the command will check that OSM can be installed and run in the
configured namespace.
`

type checkCmd struct {
	out        io.Writer
	preInstall bool
}

func newCheckCmd(out io.Writer) *cobra.Command {
	checkCmd := &checkCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "check",
		Short: "check osm control plane",
		Long:  checkDescription,
		RunE: func(*cobra.Command, []string) error {
			return checkCmd.run()
		},
	}

	f := cmd.Flags()
	f.BoolVar(&checkCmd.preInstall, "pre-install", checkCmd.preInstall, "run only pre-install checks")

	return cmd
}

func (c *checkCmd) run() error {
	checker := check.NewChecker(settings)

	var checks []check.Check

	if c.preInstall {
		checks = check.PreinstallChecks()
	}

	pass := checker.Run(checks, func(r *check.Result) {
		if r.Err != nil {
			fmt.Fprintf(c.out, "ERROR: ")
		} else {
			fmt.Fprintf(c.out, "ok: ")
		}
		fmt.Fprintln(c.out, r.Name)
		if r.Err != nil {
			fmt.Fprintf(c.out, "\t%v\n", r.Err)
		}
	})

	if !pass {
		return errors.New("Checks failed")
	}

	if len(checks) == 0 {
		fmt.Fprintln(c.out, "No checks run")
	} else {
		fmt.Fprintln(c.out, "All checks successful!")
	}
	return nil
}
