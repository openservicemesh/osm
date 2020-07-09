#!/bin/bash


# This script port forwards from the BOOKBUYER pod to local port 8081


# shellcheck disable=SC1091
source .env

backend="${1:-bookbuyer}"
thisScript="$(dirname "$0")/$(basename "$0")"

if [ -z "$backend" ]; then
    echo "Usage: $thisScript <backend-name>"
    exit 1
fi

POD="$(kubectl get pods --selector app="$backend" -n "$BOOKBUYER_NAMESPACE" --no-headers | grep 'Running' | awk '{print $1}')"

kubectl describe pod "$POD" -n "$BOOKBUYER_NAMESPACE"

kubectl port-forward "$POD" -n "$BOOKBUYER_NAMESPACE" 8081:80
