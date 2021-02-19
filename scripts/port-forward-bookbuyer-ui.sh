#!/bin/bash


# This script port forwards from the BOOKBUYER pod to local port 8080


# shellcheck disable=SC1091
source .env

BOOKBUYER_LOCAL_PORT="${BOOKBUYER_LOCAL_PORT:-8080}"
POD="$(kubectl get pods --selector app=bookbuyer -n "$BOOKBUYER_NAMESPACE" --no-headers  | grep 'Running' | awk 'NR==1{print $1}')"

kubectl port-forward "$POD" -n "$BOOKBUYER_NAMESPACE" "$BOOKBUYER_LOCAL_PORT":80
