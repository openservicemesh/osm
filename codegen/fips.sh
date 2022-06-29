#!/usr/bin/env bash

set -eu

ROOT_DIR="$(git rev-parse --show-toplevel)"

cp fips.go.txt "$ROOT_DIR/cmd/osm-bootstrap/fips.go"
cp fips.go.txt "$ROOT_DIR/cmd/osm-controller/fips.go"
cp fips.go.txt "$ROOT_DIR/cmd/osm-healthcheck/fips.go"
cp fips.go.txt "$ROOT_DIR/cmd/osm-injector/fips.go"
