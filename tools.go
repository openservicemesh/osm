// +build tools

package tools

import (
	_ "github.com/AlekSi/gocov-xml"
	_ "github.com/axw/gocov/gocov"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/jstemmer/go-junit-report"
	_ "github.com/matm/gocov-html"
	_ "github.com/mitchellh/gox"
	_ "github.com/norwoodj/helm-docs/cmd/helm-docs"
)
