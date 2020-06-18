package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/open-service-mesh/osm/pkg/check"
	"github.com/spf13/cobra"
)

const checkDescription = `
The check command checks that osm is installed and running properly.
`

type checkCmd struct {
	out io.Writer
	pre bool
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
	f.BoolVar(&checkCmd.pre, "pre", checkCmd.pre, "run only pre-install checks")

	return cmd
}

func (c *checkCmd) run() error {
	checker := check.NewChecker(settings)

	var checks []check.Check

	if c.pre {
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
