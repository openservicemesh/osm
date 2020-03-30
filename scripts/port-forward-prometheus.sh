#!/bin/bash
# shellcheck disable=SC1091
source .env

POD="$(kubectl get pods --selector app="$PROMETHEUS_SVC-server" -n "$PROMETHEUS_NAMESPACE" --no-headers | grep 'Running' | awk '{print $1}')"

kubectl port-forward "$POD" -n "$PROMETHEUS_NAMESPACE" 9090:9090

