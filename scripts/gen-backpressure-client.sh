#!/bin/bash

set -aueo pipefail

DIR="$GOPATH/src/k8s.io/"

if [ ! -d "${DIR}code-generator" ]; then
  mkdir -p "$DIR"
  pushd "$DIR"
  git clone git@github.com:kubernetes/code-generator.git
  popd
fi

../code-generator/generate-groups.sh \
    all \
    github.com/openservicemesh/osm/experimental/pkg/client \
    github.com/openservicemesh/osm/experimental/pkg/apis \
    "policy:v1alpha1" \
    --go-header-file ../code-generator/hack/boilerplate.go.txt
