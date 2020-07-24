#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

[[ $# -eq 0 ]] && { echo "Missing required namespace argument"; exit 1; }
namespace="$1"

# CI defines a DOCKER_PASS env var instead of CTR_REGISTRY_PASSWORD
if [[ "${CI:-}" == "true" ]]; then
    CTR_REGISTRY_PASSWORD=${DOCKER_PASS:-}
fi

[[ -z "$CTR_REGISTRY_PASSWORD" ]] && exit 0

REGISTRY=$(echo "$CTR_REGISTRY" | awk -F'.' '{print $1}')
REGISTRY_URL=$(echo "$CTR_REGISTRY" | awk -F'.' '{print $1 "." $2 "." $3}')

echo "Creating container registry credentials ($CTR_REGISTRY_CREDS_NAME) for Kubernetes in namespace ($namespace) for the given Azure Container Registry ($REGISTRY_URL)"

kubectl delete secrets "$CTR_REGISTRY_CREDS_NAME" -n "$namespace" --ignore-not-found
kubectl create secret docker-registry "$CTR_REGISTRY_CREDS_NAME" \
    -n "$namespace" \
    --docker-server="$REGISTRY_URL" \
    --docker-username="$REGISTRY" \
    --docker-email="noone@example.com" \
    --docker-password="$CTR_REGISTRY_PASSWORD"
