#!/bin/bash
# shellcheck disable=SC1091
source .env

selector="$1"
thisScript="$(dirname "$0")/$(basename "$0")"

if [ -z "$selector" ]; then
    echo "Usage: $thisScript <selector>"
    exit 1
fi

POD="$(kubectl get pods --selector "$selector" -n "$BOOKSTORE_NAMESPACE" --no-headers | grep 'Running' | awk 'NR==1{print $1}')"
kubectl port-forward "$POD" -n "$BOOKSTORE_NAMESPACE" 15000:15000

