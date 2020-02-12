#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

rm -rf ./certs

./demo/clean-kubernetes.sh

targets=(
    build-cert
    docker-push-cds
    docker-push-lds
    docker-push-eds
    docker-push-sds
    docker-push-rds
    docker-push-init
    docker-push-bookbuyer
    docker-push-bookstore
)

parallel --jobs 16 make ::: ${targets[@]}

# Create the proxy certificates
./demo/gen-ca.sh

./demo/create-container-registry-creds.sh
./demo/deploy-envoyproxy-config.sh

kubectl create configmap kubeconfig --from-file="$HOME/.kube/config" -n "$K8S_NAMESPACE"
kubectl create configmap azureconfig --from-file="$HOME/.azure/azureAuth.json" -n "$K8S_NAMESPACE"
kubectl apply -f crd/AzureResource.yaml
kubectl apply -f demo/AzureResource.yaml


./demo/deploy-bookbuyer.sh

# ./demo/deploy-bookstore.sh bookstore
./demo/deploy-bookstore.sh "bookstore-1"
# ./demo/deploy-bookstore.sh bookstore-2

./demo/deploy-cds.sh
./demo/deploy-sds.sh
./demo/deploy-eds.sh
./demo/deploy-rds.sh
./demo/deploy-lds.sh

./demo/deploy-traffic-split.sh
./demo/deploy-traffic-spec.sh
./demo/deploy-traffic-target.sh

watch -n0.5 "kubectl get pods -n${K8S_NAMESPACE} -o wide"
