#!/bin/bash


# This script forwards port 80 from the BOOKSTORE to local port 8081.


# shellcheck disable=SC1091
source .env

backend="${1:-bookstore-v1}"
thisScript="$(dirname "$0")/$(basename "$0")"

if [ -z "$backend" ]; then
    echo "Usage: $thisScript <backend-name>"
    exit 1
fi

BOOKSTOREv1_LOCAL_PORT="${BOOKSTOREv1_LOCAL_PORT:-8081}"
POD="$(kubectl get pods --selector app="$backend" -n "$BOOKSTORE_NAMESPACE" --no-headers | grep 'Running' | awk '{print $1}')"

kubectl port-forward "$POD" -n "$BOOKSTORE_NAMESPACE" "$BOOKSTOREv1_LOCAL_PORT":80
