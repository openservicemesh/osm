#!/bin/bash


# This script forwards BOOKSTORE port 80 to local host port 8082


# shellcheck disable=SC1091
source .env

backend="${1:-bookstore-v2}"
thisScript="$(dirname "$0")/$(basename "$0")"

if [ -z "$backend" ]; then
    echo "Usage: $thisScript <backend-name>"
    exit 1
fi

BOOKSTOREv2_LOCAL_PORT="${BOOKSTOREv2_LOCAL_PORT:-8082}"
POD="$(kubectl get pods --selector app="$backend" -n "$BOOKSTORE_NAMESPACE" --no-headers | grep 'Running' | awk '{print $1}')"

kubectl port-forward "$POD" -n "$BOOKSTORE_NAMESPACE" "$BOOKSTOREv2_LOCAL_PORT":80
