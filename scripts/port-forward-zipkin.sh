#!/bin/bash

# shellcheck disable=SC1091
set -aueo pipefail

source .env

OSM_POD=$(kubectl get pods -n "$K8S_NAMESPACE" --no-headers  --selector app=zipkin | grep Running | awk '{print $1}')

kubectl port-forward -n "$K8S_NAMESPACE" "$OSM_POD"  9411:9411
