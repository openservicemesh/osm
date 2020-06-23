#!/bin/bash

set -auexo pipefail

DIR="$GOPATH/src/k8s.io/"

if [ ! -d "${DIR}code-generator" ]; then
  mkdir -p "$DIR"
  pushd "$DIR"
  git clone git@github.com:kubernetes/code-generator.git
  popd
fi

../code-generator/generate-groups.sh \
    all \
    github.com/open-service-mesh/osm/pkg/client \
    github.com/open-service-mesh/osm/pkg/apis \
    "osmconfig:v1" \
    --go-header-file ../code-generator/hack/boilerplate.go.txt
