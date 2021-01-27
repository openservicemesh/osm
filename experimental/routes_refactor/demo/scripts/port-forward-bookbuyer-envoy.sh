#!/bin/bash

# shellcheck disable=SC1091

BOOKBUYER_LOCAL_PORT="${BOOKBUYER_LOCAL_PORT:-8080}"
POD="$(kubectl get pods --selector app=bookbuyer -n "$BOOKBUYER_NAMESPACE" --no-headers | grep 'Running' | awk '{print $1}')"

kubectl port-forward "$POD" -n "$BOOKBUYER_NAMESPACE" 15000:15000