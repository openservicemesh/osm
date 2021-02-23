#!/bin/bash

set -aueo pipefail

readarray -t modules < <(go list ./... | \
                             grep -v tests/framework | \
                             grep -v tests/e2e | \
                             grep -v tests/scenarios | \
                             grep -v tests/scale | \
                             grep -v ci/ | \
                             grep -v demo/ | \
                             grep -v experimental/ | \
                             grep -v scripts/)

go test -timeout 120s \
   -failfast \
   -v \
   -coverprofile=coverage.txt.with_generated_code \
   -covermode count "${modules[@]}" | tee testoutput.txt || { echo "go test returned non-zero"; exit 1; }

# shellcheck disable=SC2002
cat coverage.txt.with_generated_code | grep -v "_generated.go" > coverage.txt

# shellcheck disable=SC2002
cat testoutput.txt | go run github.com/jstemmer/go-junit-report > report.xml

go run github.com/axw/gocov/gocov convert coverage.txt > coverage.json

go run github.com/AlekSi/gocov-xml < coverage.json > coverage.xml

mkdir -p coverage

go run github.com/matm/gocov-html < coverage.json > coverage/index.html
