package main

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/openservicemesh/osm/pkg/errcode"
)

const errInfoDescription = `
This command lists the mapping of one or all error codes to their description.`

const errInfoExample = `
Get the description for the error code E1000
# osm support error-info E1000

Get the description for all error codes
# osm support error-info
`

type errInfoCmd struct {
	out io.Writer
}

func newSupportErrInfoCmd(out io.Writer) *cobra.Command {
	errInfoCmd := &errInfoCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "error-info",
		Short: "lists mapping of error code to its description",
		Long:  errInfoDescription,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			var errCode string
			if len(args) != 0 {
				errCode = args[0]
			}
			return errInfoCmd.run(errCode)
		},
		Example: errInfoExample,
	}

	return cmd
}

func (cmd *errInfoCmd) run(errCode string) error {
	table := tablewriter.NewWriter(cmd.out)
	table.SetHeader([]string{"Error code", "Description"})
	table.SetRowLine(true)
	table.SetColWidth(80)

	if errCode != "" {
		// Print the error code description mapping only for the given error code
		e, err := errcode.FromStr(errCode)
		if err != nil {
			return fmt.Errorf("error code '%s' is not a valid error code format, should be of the form Exxxx, ex. E1000", errCode)
		}
		description, ok := errcode.ErrCodeMap[e]
		if !ok {
			return fmt.Errorf("error code '%s' is not a valid error code recognized by OSM", errCode)
		}
		table.Append([]string{errCode, description})
	} else {
		// Print the error code description mapping for all error codes
		var sortedErrKeys []errcode.ErrCode
		for err := range errcode.ErrCodeMap {
			sortedErrKeys = append(sortedErrKeys, err)
		}
		sort.Slice(sortedErrKeys, func(i, j int) bool {
			return sortedErrKeys[i] < sortedErrKeys[j]
		})

		for _, key := range sortedErrKeys {
			desc := errcode.ErrCodeMap[key]
			desc = strings.Trim(desc, "\n") // Trim leading and trailing newlines for consistent formatting
			table.Append([]string{key.String(), desc})
		}
	}

	table.Render()

	return nil
}
