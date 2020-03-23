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

./demo/clean-kubernetes.sh

# The demo uses the following namespaces defined by environment variables:
# 1. K8S_NAMESPACE: OSM's namespace
# 2. BOOKBUYER_NAMESPACE: Namespace for the Bookbuyer service
# 3. BOOKSTORE_NAMESPACE: Namespace for the Bookstore service
kubectl create namespace "$K8S_NAMESPACE"
for ns in "$BOOKBUYER_NAMESPACE" "$BOOKSTORE_NAMESPACE" "$BOOKTHIEF_NAMESPACE"; do
    kubectl create namespace "$ns"
    kubectl label  namespaces "$ns" osm-inject="$OSM_ID"
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

# Deploy OSM
./demo/deploy-secrets.sh "ads"
./demo/deploy-webhook-secrets.sh
go run ./demo/cmd/deploy/xds.go

# Wait for POD to be ready before deploying the webhook config.
# K8s API server will probe on the webhook port when the config is deployed.
while [ "$(kubectl get pods -n "$K8S_NAMESPACE" ads -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}')" != "True" ];
do
  echo "waiting for pod ads to be ready" && sleep 2
done

./demo/deploy-webhook.sh "ads" "$K8S_NAMESPACE" "$OSM_ID"

# Deploy bookstore before applying SMI policies. The POD spec is annotated with sidecar injection as enabled.
./demo/deploy-bookstore.sh "bookstore"
./demo/deploy-bookstore.sh "bookstore-1"
./demo/deploy-bookstore.sh "bookstore-2"

# Apply SMI policies
./demo/deploy-traffic-split.sh
./demo/deploy-traffic-spec.sh
./demo/deploy-traffic-target.sh
./demo/deploy-traffic-target-2.sh
# This is a temporary workaround to have envoy run on the bookthief pod
# TODO: To remove this, annotate the POD and also update the test to not
# expect a 404. This is because if an SMI policy is not defined but sidecar
# is injected, the gRPC xDS stream will be dropped by ADS because there is
# no service/service-account associated with the service.
./demo/deploy-traffic-target-bookthief.sh

# Since the bookbuyer POD is not annotated for sidecar injection, we rely on implicit
# sidecar injection based on SMI policies by deploying bookbuyer AFTER the SMI policies
# referencing it have been applied.
./demo/deploy-bookbuyer.sh
./demo/deploy-bookthief.sh

if [[ "$IS_GITHUB" != "true" ]]; then
    watch -n5 "printf \"Namespace ${K8S_NAMESPACE}:\n\"; kubectl get pods -n ${K8S_NAMESPACE} -o wide; printf \"\n\n\"; printf \"Namespace ${BOOKBUYER_NAMESPACE}:\n\"; kubectl get pods -n ${BOOKBUYER_NAMESPACE} -o wide; printf \"\n\n\"; printf \"Namespace ${BOOKSTORE_NAMESPACE}:\n\"; kubectl get pods -n ${BOOKSTORE_NAMESPACE} -o wide; printf \"\n\n\"; printf \"Namespace ${BOOKTHIEF_NAMESPACE}:\n\"; kubectl get pods -n ${BOOKTHIEF_NAMESPACE} -o wide"
fi
