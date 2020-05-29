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

wait_for_ads_pod() {
    # Wait for POD to be ready before deploying the apps.
    ads_pod_name=$(kubectl get pods -n "$K8S_NAMESPACE" --selector app=ads --no-headers | awk '{print $1}')
    expect_status="Running"
    max=12
    for x in $(seq 1 $max); do
        pod_status="$(kubectl get pods -n "$K8S_NAMESPACE" "ads" -o 'jsonpath={..status.phase}')"
        if [ "$pod_status" == "$expect_status" ]; then
            return
        fi
        echo "[${x}] Pod status is ${pod_status}; waiting for pod ${ads_pod_name} to be $expect_status" && sleep 5
    done

    exit_error "Pod ${ads_pod_name} status is ${pod_status} -- still not $expect_status"
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
if [ -z "$CERT_MANAGER" ]; then
    exit_error "Missing CERT_MANAGER env variable"
fi
if [ -z "$CTR_REGISTRY" ]; then
    exit_error "Missing CTR_REGISTRY env variable"
fi
if [ -z "$CTR_REGISTRY_CREDS_NAME" ]; then
    exit_error "Missing CTR_REGISTRY_CREDS_NAME env variable"
fi
if [ -z "$CTR_TAG" ]; then
    exit_error "Missing CTR_TAG env variable"
fi
if [ -z "$PROMETHEUS_SVC" ]; then
    exit_error "Missing PROMETHEUS_SVC env variable"
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
wait_for_ads_pod

./demo/deploy-apps.sh


if [[ "$IS_GITHUB" != "true" ]]; then
    watch -n5 "printf \"Namespace ${K8S_NAMESPACE}:\n\"; kubectl get pods -n ${K8S_NAMESPACE} -o wide; printf \"\n\n\"; printf \"Namespace ${BOOKBUYER_NAMESPACE}:\n\"; kubectl get pods -n ${BOOKBUYER_NAMESPACE} -o wide; printf \"\n\n\"; printf \"Namespace ${BOOKSTORE_NAMESPACE}:\n\"; kubectl get pods -n ${BOOKSTORE_NAMESPACE} -o wide; printf \"\n\n\"; printf \"Namespace ${BOOKTHIEF_NAMESPACE}:\n\"; kubectl get pods -n ${BOOKTHIEF_NAMESPACE} -o wide"
fi
