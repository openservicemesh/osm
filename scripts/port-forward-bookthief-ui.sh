#!/bin/bash


# This script forwards port 80 from the BOOKTHIEF pod to local port 8083


# shellcheck disable=SC1091
source .env

BOOKTHIEF_LOCAL_PORT="${BOOKTHIEF_LOCAL_PORT:-8083}"
POD="$(kubectl get pods --selector app=bookthief -n "$BOOKTHIEF_NAMESPACE" --no-headers | grep 'Running' | awk '{print $1}')"

kubectl port-forward "$POD" -n "$BOOKTHIEF_NAMESPACE" "$BOOKTHIEF_LOCAL_PORT":80
