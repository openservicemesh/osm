#!/bin/bash

set -auexo pipefail

# shellcheck disable=SC1091
source .env

rm -rf ./certificates
rm -rf ./certs

./demo/clean-kubernetes.sh

echo "Create the bootstrap.yaml file"
./demo/create-bootstrap-yaml.sh

make build-cert
make docker-push-ads
make docker-push-init
make docker-push-envoyproxy
make docker-push-bookbuyer
make docker-push-bookstore

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
./demo/deploy-envoyproxy-config.sh


kubectl apply -f crd/AzureResource.yaml
./demo/deploy-AzureResource.sh

./demo/deploy-bookbuyer.sh

./demo/deploy-bookstore.sh bookstore
./demo/deploy-bookstore.sh bookstore-1
./demo/deploy-bookstore.sh bookstore-2

./demo/deploy-xds.sh

./demo/deploy-traffic-split.sh
./demo/deploy-traffic-spec.sh
./demo/deploy-traffic-target.sh

if [[ "$IS_GITHUB" != "true" ]]; then
    watch -n0.5 "kubectl get pods -n${K8S_NAMESPACE} -o wide"
fi
