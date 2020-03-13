#!/bin/bash

set -auexo pipefail

# shellcheck disable=SC1091
source .env
IS_GITHUB="${IS_GITHUB:-default false}"

rm -rf ./certificates
rm -rf ./certs

./demo/clean-kubernetes.sh
go run  demo/cmd/bootstrap/create.go

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

./demo/deploy-bookstore.sh "bookstore"
./demo/deploy-bookstore.sh "bookstore-1"
./demo/deploy-bookstore.sh "bookstore-2"

if [[ "$IS_GITHUB" != "true" ]]; then
    watch -n0.5 "kubectl get pods -n${K8S_NAMESPACE} -o wide"
fi
