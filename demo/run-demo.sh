#!/bin/bash

set -auexo pipefail

# shellcheck disable=SC1091
source .env
IS_GITHUB="${IS_GITHUB:-default false}"

rm -rf ./certificates
rm -rf ./certs

exit_error() {
    error="$1"
    echo "$error"
    exit 1
}

# Check for required environment variables
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

./demo/clean-kubernetes.sh

# The demo uses the following namespaces defined by environment variables:
# 1. K8S_NAMESPACE: OSM's namespace
# 2. BOOKBUYER_NAMESPACE: Namespace for the Bookbuyer service
# 3. BOOKSTORE_NAMESPACE: Namespace for the Bookstore service
kubectl create namespace "$K8S_NAMESPACE"
for ns in "$BOOKBUYER_NAMESPACE" "$BOOKSTORE_NAMESPACE" "$BOOKTHIEF_NAMESPACE"; do
    kubectl create namespace "$ns"
    kubectl label  namespaces "$ns" osm-inject="enabled"
done
# APP_NAMESPACES is a comma separated list of namespaces that informs OSM of the
# namespaces it should observe.
export APP_NAMESPACES="$BOOKBUYER_NAMESPACE,$BOOKSTORE_NAMESPACE,$BOOKTHIEF_NAMESPACE"

make build-cert

./demo/build-push-images.sh

# Create the proxy certificates
./demo/gen-ca.sh

if [[ "$IS_GITHUB" != "true" ]]; then
    ./demo/create-container-registry-creds.sh
else
    mkdir -p "$HOME/.kube"
    touch "$HOME/.kube/config"
    mkdir -p "$HOME/.azure"
    touch "$HOME/.azure/azureAuth.json"
    # This script is specifically for CI
    ./ci/create-container-registry-creds.sh
fi

kubectl create configmap kubeconfig  --from-file="$HOME/.kube/config"          -n "$K8S_NAMESPACE"
kubectl create configmap azureconfig --from-file="$HOME/.azure/azureAuth.json" -n "$K8S_NAMESPACE"

kubectl apply -f crd/AzureResource.yaml
./demo/deploy-AzureResource.sh

./demo/deploy-traffic-split.sh
./demo/deploy-traffic-spec.sh
./demo/deploy-traffic-target.sh
./demo/deploy-traffic-target-2.sh
# this is a temporary workaround to have envoy run on the bookthief pod
# todo: remove this once we have annotations supported
./demo/deploy-traffic-target-bookthief.sh

./demo/deploy-secrets.sh "ads"
./demo/deploy-webhook-secrets.sh
go run ./demo/cmd/deploy/xds.go

# Wait for POD to be ready before
while [ "$(kubectl get pods -n "$K8S_NAMESPACE" ads -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}')" != "True" ];
do
  echo "waiting for pod ads to be ready" && sleep 2
done

./demo/deploy-webhook.sh "ads" "$K8S_NAMESPACE"

# The POD creation for the services will fail if OSM has not picked up the
# corresponding services defined in the SMI spec
./demo/deploy-bookbuyer.sh
./demo/deploy-bookthief.sh

./demo/deploy-bookstore.sh "bookstore"
./demo/deploy-bookstore.sh "bookstore-1"
./demo/deploy-bookstore.sh "bookstore-2"

if [[ "$IS_GITHUB" != "true" ]]; then
    watch -n5 "printf \"Namespace ${K8S_NAMESPACE}:\n\"; kubectl get pods -n ${K8S_NAMESPACE} -o wide; printf \"\n\n\"; printf \"Namespace ${BOOKBUYER_NAMESPACE}:\n\"; kubectl get pods -n ${BOOKBUYER_NAMESPACE} -o wide; printf \"\n\n\"; printf \"Namespace ${BOOKSTORE_NAMESPACE}:\n\"; kubectl get pods -n ${BOOKSTORE_NAMESPACE} -o wide"
fi
