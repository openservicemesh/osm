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

POD="$(kubectl get pods --selector app="$backend" -n "$BOOKSTORE_NAMESPACE" --no-headers | grep 'Running' | awk '{print $1}')"

kubectl describe pod "$POD" -n "$BOOKSTORE_NAMESPACE"

# Check if run as standalone or from scripts/port-forward-all.sh
if [ $# -eq 0 ]; then
  # Prompt for a port number
  read -p "Enter BOOKSTORE pod local port:" -i "8082" -e PORT
else
  # Port number read from scripts/port-forward-all.sh
  PORT=$2
fi

kubectl port-forward "$POD" -n "$BOOKSTORE_NAMESPACE" $PORT:80
