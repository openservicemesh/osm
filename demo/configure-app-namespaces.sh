#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

for ns in "$BOOKWAREHOUSE_NAMESPACE" "$BOOKBUYER_NAMESPACE" "$BOOKSTORE_NAMESPACE" "$BOOKTHIEF_NAMESPACE"; do
    kubectl create namespace "$ns" --save-config
done

if [[ "${CI:-}" != "true" ]]; then
  REGISTRY=$(echo "$CTR_REGISTRY" | awk -F'.' '{print $1}')
  REGISTRY_URL=$(echo "$CTR_REGISTRY" | awk -F'.' '{print $1 "." $2 "." $3}')

  for ns in "$BOOKWAREHOUSE_NAMESPACE" "$BOOKBUYER_NAMESPACE" "$BOOKSTORE_NAMESPACE" "$BOOKTHIEF_NAMESPACE"; do
      kubectl delete secrets "$CTR_REGISTRY_CREDS_NAME" -n "$ns" || true
      kubectl create secret docker-registry "$CTR_REGISTRY_CREDS_NAME" \
          -n "$ns" \
          --docker-server="$REGISTRY_URL" \
          --docker-username="$REGISTRY" \
          --docker-email="noone@example.com" \
          --docker-password="$CTR_REGISTRY_PASSWORD"
  done
else
    # This script is specifically for CI
    ./ci/create-app-container-registry-creds.sh
fi
