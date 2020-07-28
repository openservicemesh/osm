#!/bin/bash

# shellcheck disable=SC1091
source .env

kubectl describe pod "$(kubectl get pods -n "$BOOKWAREHOUSE_NAMESPACE" --show-labels --selector app=client --no-headers | grep -v 'Terminating' | awk '{print $1}' | head -n1)" -n "$BOOKWAREHOUSE_NAMESPACE"

POD="$(kubectl get pods -n "$BOOKWAREHOUSE_NAMESPACE" --show-labels --selector app=bookwarehouse --no-headers | grep -v 'Terminating' | awk '{print $1}' | head -n1)"

kubectl logs "${POD}" -n "$BOOKWAREHOUSE_NAMESPACE" -c envoy --tail=100 -f
