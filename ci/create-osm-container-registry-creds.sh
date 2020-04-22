#!/bin/bash
set -aueo pipefail

REGISTRY=$(echo "$CTR_REGISTRY" | awk -F'.' '{print $1}')
REGISTRY_URL=$(echo "$CTR_REGISTRY" | awk -F'.' '{print $1 "." $2 "." $3}')

kubectl delete secrets "$CTR_REGISTRY_CREDS_NAME" -n "$K8S_NAMESPACE" || true
kubectl create secret docker-registry "$CTR_REGISTRY_CREDS_NAME" \
    -n "$K8S_NAMESPACE" \
    --docker-server="$REGISTRY_URL" \
    --docker-username="$REGISTRY" \
    --docker-email="noone@example.com" \
    --docker-password="$DOCKER_PASS"
