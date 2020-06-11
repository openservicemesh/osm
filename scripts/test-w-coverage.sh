#!/bin/bash

set -aueo pipefail

GOPATH=$HOME/go
GOBIN="$(go env GOPATH)/bin"
PATH=$PATH:$GOPATH
PATH=$PATH:$GOBIN

## Prereqs
# go get -u golang.org/x/lint/golint
# go get -u github.com/jstemmer/go-junit-report
# go get -u github.com/axw/gocov/gocov
# go get -u github.com/AlekSi/gocov-xml
# go get -u github.com/matm/gocov-html

go test -timeout 80s \
   -v \
   -coverprofile=coverage.txt \
   -covermode count ./... | tee testoutput.txt || { echo "go test returned non-zero"; cat testoutput.txt; exit 1; }

# shellcheck disable=SC2002
cat testoutput.txt | go-junit-report > report.xml

gocov convert coverage.txt > coverage.json

gocov-xml < coverage.json > coverage.xml

mkdir -p coverage

gocov-html < coverage.json > coverage/index.html
