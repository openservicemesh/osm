#!/bin/bash

set -auexo pipefail

# shellcheck disable=SC1091
source .env

for ns in "$BOOKWAREHOUSE_NAMESPACE" "$BOOKBUYER_NAMESPACE" "$BOOKSTORE_NAMESPACE" "$BOOKTHIEF_NAMESPACE"; do
    kubectl create namespace "$ns"
    kubectl label  namespaces "$ns" openservicemesh.io/monitor="$OSM_ID"
done

if [[ "$IS_GITHUB" != "true" ]]; then
  REGISTRY=$(echo "$CTR_REGISTRY" | awk -F'.' '{print $1}')
  REGISTRY_URL=$(echo "$CTR_REGISTRY" | awk -F'.' '{print $1 "." $2 "." $3}')

  DOCKER_PASSWORD=$(az acr credential show -n "$REGISTRY" --query "passwords[0].value" | tr -d '"')

  for ns in "$BOOKWAREHOUSE_NAMESPACE" "$BOOKBUYER_NAMESPACE" "$BOOKSTORE_NAMESPACE" "$BOOKTHIEF_NAMESPACE"; do
      kubectl delete secrets "$CTR_REGISTRY_CREDS_NAME" -n "$ns" || true
      kubectl create secret docker-registry "$CTR_REGISTRY_CREDS_NAME" \
          -n "$ns" \
          --docker-server="$REGISTRY_URL" \
          --docker-username="$REGISTRY" \
          --docker-email="noone@example.com" \
          --docker-password="$DOCKER_PASSWORD"
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

# Apply SMI policies
./demo/deploy-traffic-split.sh
./demo/deploy-traffic-spec.sh
./demo/deploy-traffic-target.sh
