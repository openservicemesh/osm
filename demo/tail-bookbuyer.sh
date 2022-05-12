#!/bin/bash

# shellcheck disable=SC1091
source .env

POD="$(kubectl get pods -n "$BOOKBUYER_NAMESPACE" --show-labels --selector app=bookbuyer --no-headers | grep -v 'Terminating' | awk '{print $1}' | head -n1)"

kubectl logs "${POD}" -n "$BOOKBUYER_NAMESPACE" -c bookbuyer --tail=100 -f
