#!/bin/bash

set -auexo pipefail

touch .env

rm -rf ./certificates
rm -rf ./certs

./demo/clean-kubernetes.sh

make build-cert

make docker-push-ads

make docker-push-init
make docker-push-bookbuyer
make docker-push-bookstore

# Create the proxy certificates
./demo/gen-ca.sh
./demo/deploy-envoyproxy-config.sh

kubectl apply -f crd/AzureResource.yaml
kubectl apply -f demo/AzureResource.yaml

./demo/deploy-bookbuyer.sh 15

./demo/deploy-bookstore.sh bookstore
./demo/deploy-bookstore.sh bookstore-1
./demo/deploy-bookstore.sh bookstore-2

./demo/deploy-xds.sh

./demo/deploy-traffic-split.sh
./demo/deploy-traffic-spec.sh
./demo/deploy-traffic-target.sh
