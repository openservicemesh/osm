#!/bin/bash

set -auexo pipefail

touch .env

# Create the proxy certificates
./scripts/gen-proxy-certificate.sh

make docker-push-cds
make docker-push-lds
make docker-push-eds
make docker-push-sds
make docker-push-rds

make docker-push-init
make docker-push-bookbuyer
make docker-push-bookstore

./demo/clean-kubernetes.sh
./demo/create-certificates.sh
./demo/deploy-certificates-config.sh
./demo/deploy-envoyproxy-config.sh

# kubectl create configmap kubeconfig --from-file="$HOME/.kube/config" -n "$K8S_NAMESPACE"
# kubectl create configmap azureconfig --from-file="$HOME/.azure/azureAuth.json" -n "$K8S_NAMESPACE"
kubectl apply -f crd/AzureResource.yaml
kubectl apply -f demo/AzureResource.yaml

./demo/deploy-bookbuyer.sh

./demo/deploy-bookstore.sh bookstore
./demo/deploy-bookstore.sh bookstore-1
./demo/deploy-bookstore.sh bookstore-2

./demo/deploy-secrets.sh

./demo/deploy-cds.sh
./demo/deploy-sds.sh
./demo/deploy-eds.sh
./demo/deploy-rds.sh
./demo/deploy-lds.sh

./demo/deploy-traffic-split.sh
./demo/deploy-traffic-spec.sh
./demo/deploy-traffic-target.sh

