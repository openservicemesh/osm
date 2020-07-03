#!/bin/bash
# shellcheck disable=SC1091
source .env

POD="$(kubectl get pods --selector app=bookbuyer -n "$BOOKBUYER_NAMESPACE" --no-headers | grep 'Running' | awk '{print $1}')"

kubectl port-forward "$POD" -n "$BOOKBUYER_NAMESPACE" 15000:15000

