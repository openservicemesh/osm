#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

for ns in "$BOOKWAREHOUSE_NAMESPACE" "$BOOKBUYER_NAMESPACE" "${BOOKSTORE_NAMESPACE}-v1" "${BOOKSTORE_NAMESPACE}-v2" "$BOOKTHIEF_NAMESPACE"; do
    kubectl create namespace "$ns" --save-config
    ./scripts/create-container-registry-creds.sh "$ns"
done
