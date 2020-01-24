#!/bin/bash

set -aueo pipefail

source .env

make docker-push-cds
make docker-push-eds
make docker-push-sds

make docker-push-init
make docker-push-bookbuyer
make docker-push-bookstore

./demo/clean-kubernetes.sh
./demo/create-container-registry-creds.sh
./demo/create-certificates.sh
./demo/deploy-certificates-config.sh
./demo/deploy-envoyproxy-config.sh

kubectl create configmap kubeconfig --from-file="$HOME/.kube/config" -n "$K8S_NAMESPACE"
kubectl create configmap azureconfig --from-file="$HOME/.azure/azureAuth.json" -n "$K8S_NAMESPACE"

./demo/deploy-bookbuyer.sh

./demo/deploy-bookstore.sh bookstore
./demo/deploy-bookstore.sh bookstore-1
./demo/deploy-bookstore.sh bookstore-2

./demo/deploy-cds.sh
./demo/deploy-sds.sh
./demo/deploy-eds.sh

./demo/deploy-traffic-split.sh

watch -n0.5 "kubectl get pods -nsmc -o wide"
