#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

kubectl delete namespace "$K8S_NAMESPACE" || true
kubectl create namespace "$K8S_NAMESPACE" || true
