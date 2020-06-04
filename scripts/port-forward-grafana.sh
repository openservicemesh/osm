#!/bin/bash
# shellcheck disable=SC1091
source .env

POD="$(kubectl get pods --selector app="osm-grafana" -n "$K8S_NAMESPACE" --no-headers | grep 'Running' | awk '{print $1}')"

kubectl port-forward "$POD" -n "$K8S_NAMESPACE" 3000:3000
