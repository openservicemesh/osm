#!/bin/bash


# This script port forwards from the BOOKBUYER pod to local port 8081


# shellcheck disable=SC1091
source .env

POD="$(kubectl get pods --selector app=bookbuyer -n "$BOOKBUYER_NAMESPACE" --no-headers | grep 'Running' | awk '{print $1}')"

kubectl describe pod "$POD" -n "$BOOKBUYER_NAMESPACE"

kubectl port-forward "$POD" -n "$BOOKBUYER_NAMESPACE" 8081:80
