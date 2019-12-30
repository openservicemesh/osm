#!/bin/bash

set -aueo pipefail

source .env

kubectl delete configmap certificates-config -n "$K8S_NAMESPACE" || true

kubectl create configmap certificates-config --from-file="$(pwd)/certificates/" -n "$K8S_NAMESPACE"
