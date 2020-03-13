#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

kubectl delete mutatingwebhookconfiguration ads --ignore-not-found=true

for NAME in bookbuyer bookstore osm; do
  kubectl delete namespace "${K8S_NAMESPACE}-${NAME}" || true
  kubectl create namespace "${K8S_NAMESPACE}-${NAME}" || true
done

kubectl delete clusterrole smc-xds || true
kubectl delete clusterrolebinding smc-xds || true
