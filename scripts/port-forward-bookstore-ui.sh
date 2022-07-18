#!/bin/bash


# This script forwards port 14001 from the BOOKSTORE to local port 8081.


# shellcheck disable=SC1091
source .env

selector="$1"
thisScript="$(dirname "$0")/$(basename "$0")"

if [ -z "$selector" ]; then
    echo "Usage: $thisScript <selector>"
    exit 1
fi

BOOKSTORE_LOCAL_PORT="${BOOKSTORE_LOCAL_PORT:-8084}"
POD="$(kubectl get pods --selector "$selector" -n "$BOOKSTORE_NAMESPACE" --no-headers | grep 'Running' | awk 'NR==1{print $1}')"

kubectl port-forward "$POD" -n "$BOOKSTORE_NAMESPACE" "$BOOKSTORE_LOCAL_PORT":14001
