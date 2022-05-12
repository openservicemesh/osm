#!/bin/bash

# shellcheck disable=SC1091
source .env

kubectl describe pod "$(kubectl get pods -n "$BOOKBUYER_NAMESPACE" --show-labels --selector app=client --no-headers | grep -v 'Terminating' | awk '{print $1}' | head -n1)" -n "$BOOKBUYER_NAMESPACE"

POD="$(kubectl get pods -n "$BOOKBUYER_NAMESPACE" --show-labels --selector app=bookbuyer --no-headers | grep -v 'Terminating' | awk '{print $1}' | head -n1)"

kubectl logs "${POD}" -n "$BOOKBUYER_NAMESPACE" -c envoy --tail=100 -f
