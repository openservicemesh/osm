#!/bin/bash


# This script forwards port 80 from the BOOKTHIEF pod to local port 8083


# shellcheck disable=SC1091
source .env

POD="$(kubectl get pods --selector app=bookthief -n "$BOOKTHIEF_NAMESPACE" --no-headers | grep 'Running' | awk '{print $1}')"

kubectl describe pod "$POD" -n "$BOOKTHIEF_NAMESPACE"

# Check if run as standalone or from scripts/port-forward-all.sh
if [ $# -eq 0 ]; then
  # Prompt for a port number
  read -p "Enter BOOKTHIEF pod local port:" -i "8083" -e PORT
else
  # Port number read from scripts/port-forward-all.sh
  PORT=$1
fi

kubectl port-forward "$POD" -n "$BOOKTHIEF_NAMESPACE" $PORT:80
