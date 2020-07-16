#!/bin/bash


# This script forwards port 80 from the BOOKTHIEF pod to local port 8083


# shellcheck disable=SC1091
source .env

POD="$(kubectl get pods --selector app=bookthief -n "$BOOKTHIEF_NAMESPACE" --no-headers | grep 'Running' | awk '{print $1}')"

kubectl describe pod "$POD" -n "$BOOKTHIEF_NAMESPACE"

kubectl port-forward "$POD" -n "$BOOKTHIEF_NAMESPACE" 8083:8080
