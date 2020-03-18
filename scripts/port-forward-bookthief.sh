#!/bin/bash
# shellcheck disable=SC1091
source .env

POD="$(kubectl get pods --selector app=bookthief -n "$BOOKTHIEF_NAMESPACE" --no-headers | grep 'Running' | awk '{print $1}')"

kubectl port-forward "$POD" -n "$BOOKTHIEF_NAMESPACE" 15000:15000

