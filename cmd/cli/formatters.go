package main

import (
	"io"
	"text/tabwriter"
)

const (
	minwidth = 6
	tabwidth = 4
	padding  = 3
	padchar  = ' '
	flags    = 0
)

func newTabWriter(out io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(out, minwidth, tabwidth, padding, padchar, flags)
}
