#!/bin/bash

# shellcheck disable=SC1091

BOOKTHIEF_LOCAL_PORT="${BOOKTHIEF_LOCAL_PORT:-8083}"
POD="$(kubectl get pods --selector app=bookthief -n "$BOOKTHIEF_NAMESPACE" --no-headers | grep 'Running' | awk '{print $1}')"

kubectl port-forward "$POD" -n "$BOOKTHIEF_NAMESPACE" 15003:15000