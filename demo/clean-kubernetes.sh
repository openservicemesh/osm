#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

kubectl delete mutatingwebhookconfiguration ads --ignore-not-found=true
kubectl delete namespace "$K8S_NAMESPACE" || true
kubectl create namespace "$K8S_NAMESPACE" || true
kubectl delete clusterrole smc-xds || true
kubectl delete clusterrolebinding smc-xds || true
