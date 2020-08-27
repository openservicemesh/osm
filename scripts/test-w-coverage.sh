#!/bin/bash

set -aueo pipefail

go test -timeout 80s \
   -v \
   -coverprofile=coverage.txt \
   -covermode count ./... | tee testoutput.txt || { echo "go test returned non-zero"; exit 1; }

# shellcheck disable=SC2002
cat testoutput.txt | go run github.com/jstemmer/go-junit-report > report.xml

go run github.com/axw/gocov/gocov convert coverage.txt > coverage.json

go run github.com/AlekSi/gocov-xml < coverage.json > coverage.xml

mkdir -p coverage

go run github.com/matm/gocov-html < coverage.json > coverage/index.html
