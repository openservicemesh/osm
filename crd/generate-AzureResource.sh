#!/bin/bash

"$GOPATH/src/k8s.io/code-generator/generate-groups.sh" \
    all \
    github.com/openservicemesh/osm/pkg/osm_client \
    github.com/openservicemesh/osm/pkg/apis \
    "azureresource:v1"
