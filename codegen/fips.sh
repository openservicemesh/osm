#!/usr/bin/env bash

set -eu

ROOT_DIR="$(git rev-parse --show-toplevel)"

cp "${ROOT_DIR}"/codegen/fips.go.txt "$ROOT_DIR/cmd/osm-bootstrap/fips.go"
cp "${ROOT_DIR}"/codegen/fips.go.txt "$ROOT_DIR/cmd/osm-controller/fips.go"
cp "${ROOT_DIR}"/codegen/fips.go.txt "$ROOT_DIR/cmd/osm-healthcheck/fips.go"
cp "${ROOT_DIR}"/codegen/fips.go.txt "$ROOT_DIR/cmd/osm-injector/fips.go"
