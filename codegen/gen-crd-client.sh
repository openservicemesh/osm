#!/usr/bin/env bash

# Script to generate client-go types and code for OSM's CRDs
#
# Copyright 2020 Open Service Mesh Authors.
#
#    Licensed under the Apache License, Version 2.0 (the "License");
#    you may not use this file except in compliance with the License.
#    You may obtain a copy of the License at
#
#        http://www.apache.org/licenses/LICENSE-2.0
#
#    Unless required by applicable law or agreed to in writing, software
#    distributed under the License is distributed on an "AS IS" BASIS,
#    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#    See the License for the specific language governing permissions and
#    limitations under the License.
#
# Copyright SMI SDK for Go authors.
#
#    Licensed under the Apache License, Version 2.0 (the "License");
#    you may not use this file except in compliance with the License.
#    You may obtain a copy of the License at
#
#        http://www.apache.org/licenses/LICENSE-2.0
#
#    Unless required by applicable law or agreed to in writing, software
#    distributed under the License is distributed on an "AS IS" BASIS,
#    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#    See the License for the specific language governing permissions and
#    limitations under the License.
#
# shellcheck disable=SC2006,SC2046,SC2116,SC2086

set -eu

ROOT_PACKAGE="github.com/openservicemesh/osm"
ROOT_DIR="$(git rev-parse --show-toplevel)"

# get code-generator version from go.sum
CODEGEN_VERSION="v0.22.1" # Must match k8s.io/client-go version defined in go.mod
CODEGEN_PKG="$(echo `go env GOPATH`/pkg/mod/k8s.io/code-generator@${CODEGEN_VERSION})"

echo ">>> using codegen: ${CODEGEN_PKG}"
# ensure we can execute the codegen script
chmod +x ${CODEGEN_PKG}/generate-groups.sh

function generate_client() {
  TEMP_DIR=$(mktemp -d)
  CUSTOM_RESOURCE_NAME=$1
  CUSTOM_RESOURCE_VERSIONS=$2

  # delete the generated deepcopy
  for V in ${CUSTOM_RESOURCE_VERSIONS//,/ }; do
    rm -f ${ROOT_DIR}/pkg/apis/${CUSTOM_RESOURCE_NAME}/${V}/zz_generated.deepcopy.go
  done

  # delete the generated code as this is additive, removed objects will not be cleaned
  rm -rf ${ROOT_DIR}/pkg/gen/client/${CUSTOM_RESOURCE_NAME}

   # code-generator makes assumptions about the project being located in `$GOPATH/src`.
  # To work around this we create a temporary directory, use it as output base and copy everything back once generated.
  "${CODEGEN_PKG}"/generate-groups.sh all \
    "$ROOT_PACKAGE/pkg/gen/client/$CUSTOM_RESOURCE_NAME" \
    "$ROOT_PACKAGE/pkg/apis" \
    $CUSTOM_RESOURCE_NAME:$CUSTOM_RESOURCE_VERSIONS \
    --go-header-file "${ROOT_DIR}"/codegen/boilerplate.go.txt \
    --output-base "${TEMP_DIR}"

  cp -r "${TEMP_DIR}/${ROOT_PACKAGE}/." "${ROOT_DIR}/"
  rm -rf ${TEMP_DIR}
}

echo "##### Generating config.openservicemesh.io client ######"
generate_client "config" "v1alpha1,v1alpha2"

echo "##### Generating policy.openservicemesh.io client ######"
generate_client "policy" "v1alpha1"
