#!/bin/bash


# This script port forwards from the BOOKBUYER pod to local port 8080


# shellcheck disable=SC1091
source .env

POD="$(kubectl get pods --selector app=bookbuyer -n "$BOOKBUYER_NAMESPACE" --no-headers | grep 'Running' | awk '{print $1}')"

kubectl describe pod "$POD" -n "$BOOKBUYER_NAMESPACE"

# Check if run as standalone or from scripts/port-forward-all.sh
if [ $# -eq 0 ]; then
  # Prompt for a port number
  read -p "Enter BOOKBUYER pod local port:" -i "8080" -e PORT
else
  # Port number read from scripts/port-forward-all.sh
  PORT=$1
fi

kubectl port-forward "$POD" -n "$BOOKBUYER_NAMESPACE" $PORT:80
