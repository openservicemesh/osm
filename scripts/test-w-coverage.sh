#!/bin/bash

set -aueo pipefail

readarray -t modules < <(go list ./... | grep -v tests/e2e | grep -v tests/scenarios | grep -v tests/scale)

go test -timeout 120s \
   -failfast \
   -v \
   -coverprofile=coverage.txt \
   -covermode count "${modules[@]}" | tee testoutput.txt || { echo "go test returned non-zero"; exit 1; }

# shellcheck disable=SC2002
cat testoutput.txt | go run github.com/jstemmer/go-junit-report > report.xml

go run github.com/axw/gocov/gocov convert coverage.txt > coverage.json

go run github.com/AlekSi/gocov-xml < coverage.json > coverage.xml

mkdir -p coverage

go run github.com/matm/gocov-html < coverage.json > coverage/index.html
