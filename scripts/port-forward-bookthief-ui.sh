#!/bin/bash


# This script forwards port 80 from the BOOKTHIEF pod to local port 8082


# shellcheck disable=SC1091
source .env

backend="${1:-bookthief}"
thisScript="$(dirname "$0")/$(basename "$0")"

if [ -z "$backend" ]; then
    echo "Usage: $thisScript <backend-name>"
    exit 1
fi

POD="$(kubectl get pods --selector app="$backend" -n "$BOOKTHIEF_NAMESPACE" --no-headers | grep 'Running' | awk '{print $1}')"

kubectl describe pod "$POD" -n "$BOOKTHIEF_NAMESPACE"

kubectl port-forward "$POD" -n "$BOOKTHIEF_NAMESPACE" 8082:80
