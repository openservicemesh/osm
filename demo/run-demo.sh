#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env
IS_GITHUB="${IS_GITHUB:-default false}"

exit_error() {
    error="$1"
    echo "$error"
    exit 1
}

# Check for required environment variables
if [ -z "$OSM_ID" ]; then
    exit_error "Missing OSM_ID env variable"
fi
if [ -z "$K8S_NAMESPACE" ]; then
    exit_error "Missing K8S_NAMESPACE env variable"
fi
if [ -z "$BOOKBUYER_NAMESPACE" ]; then
    exit_error "Missing BOOKBUYER_NAMESPACE env variable"
fi
if [ -z "$BOOKSTORE_NAMESPACE" ]; then
    exit_error "Missing BOOKSTORE_NAMESPACE env variable"
fi
if [ -z "$BOOKTHIEF_NAMESPACE" ]; then
    exit_error "Missing BOOKTHIEF_NAMESPACE env variable"
fi

if [[ "$IS_GITHUB" != "true" ]]; then
    # In Github CI we always use a new namespace - so this is not necessary
    ./demo/clean-kubernetes.sh
fi

# The demo uses osm's namespace as defined by environment variables, K8S_NAMESPACE
# to house the control plane components.
kubectl create namespace "$K8S_NAMESPACE"

echo "Certificate Manager in use: $CERT_MANAGER"
if [ "$CERT_MANAGER" = "vault" ]; then
    echo "Installing Hashi Vault"
    ./demo/deploy-vault.sh
fi

if [[ "$IS_GITHUB" != "true" ]]; then
    # For Github CI we achieve these at a different time or different script
    # See .github/workflows/main.yml
    ./demo/build-push-images.sh
    ./demo/create-container-registry-creds.sh
else
    mkdir -p "$HOME/.kube"
    touch "$HOME/.kube/config"
    mkdir -p "$HOME/.azure"
    touch "$HOME/.azure/azureAuth.json"
    # This script is specifically for CI
    ./ci/create-osm-container-registry-creds.sh
fi

kubectl create configmap kubeconfig  --from-file="$HOME/.kube/config"          -n "$K8S_NAMESPACE"
kubectl create configmap azureconfig --from-file="$HOME/.azure/azureAuth.json" -n "$K8S_NAMESPACE"

kubectl apply -f crd/AzureResource.yaml
./demo/deploy-AzureResource.sh

# Deploys Xds and Prometheus
go run ./demo/cmd/deploy/control-plane.go

# Wait for POD to be ready before deploying the apps.
while [ "$(kubectl get pods -n "$K8S_NAMESPACE" ads -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}')" != "True" ];
do
  echo "waiting for pod ads to be ready" && sleep 5
done

./demo/deploy-apps.sh


if [[ "$IS_GITHUB" != "true" ]]; then
    watch -n5 "printf \"Namespace ${K8S_NAMESPACE}:\n\"; kubectl get pods -n ${K8S_NAMESPACE} -o wide; printf \"\n\n\"; printf \"Namespace ${BOOKBUYER_NAMESPACE}:\n\"; kubectl get pods -n ${BOOKBUYER_NAMESPACE} -o wide; printf \"\n\n\"; printf \"Namespace ${BOOKSTORE_NAMESPACE}:\n\"; kubectl get pods -n ${BOOKSTORE_NAMESPACE} -o wide; printf \"\n\n\"; printf \"Namespace ${BOOKTHIEF_NAMESPACE}:\n\"; kubectl get pods -n ${BOOKTHIEF_NAMESPACE} -o wide"
fi
