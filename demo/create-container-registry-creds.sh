#!/bin/bash

set -aueo pipefail

source .env

REGISTRY="draychev"
REGISTRY_URL="draychev.azurecr.io"
CREDS_NAME="acr-creds"

echo "Creating container registry credentials ($CREDS_NAME) for Kubernetes for the given Azure Container Registry ($REGISTRY_URL)"

DOCKER_PASSWORD=$(az acr credential show -n $REGISTRY --query "passwords[0].value" | tr -d '"')

kubectl delete secrets acr-creds -n diplomat || true
kubectl create secret docker-registry "$CREDS_NAME" \
        -n "$K8S_NAMESPACE" \
        --docker-server="$REGISTRY_URL" \
        --docker-username="$REGISTRY" \
        --docker-email="noone@example.com" \
        --docker-password="$DOCKER_PASSWORD"
