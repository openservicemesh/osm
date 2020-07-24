#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

if [[ "${IS_GITHUB:-}" != "true" ]]; then
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
# Deploy bookstore
./demo/deploy-bookstore.sh "v1"
./demo/deploy-bookstore.sh "v2"
# Deploy bookbuyer
./demo/deploy-bookbuyer.sh
# Deploy bookthief
./demo/deploy-bookthief.sh

./demo/deploy-bookwarehouse.sh
