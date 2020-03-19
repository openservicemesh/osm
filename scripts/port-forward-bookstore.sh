#!/bin/bash
# shellcheck disable=SC1091
source .env

backend="$1"
thisScript="$(dirname "$0")/$(basename "$0")"

if [ -z "$backend" ]; then
    echo "Usage: $thisScript <backend-name>"
    exit 1
fi

POD="$(kubectl get pods --selector app="$backend" -n "$BOOKSTORE_NAMESPACE" --no-headers | grep 'Running' | awk '{print $1}')"
kubectl port-forward "$POD" -n "$BOOKSTORE_NAMESPACE" 15000:15000

