#!/bin/bash

"$GOPATH/src/k8s.io/code-generator/generate-groups.sh" \
    all \
    github.com/open-service-mesh/osm/pkg/osm_client \
    github.com/open-service-mesh/osm/pkg/apis \
    "azureresource:v1"
