#!/bin/bash

# shellcheck disable=SC1091
source .env

POD="$(kubectl get pods -n "$BOOKWAREHOUSE_NAMESPACE" --show-labels --selector app=bookwarehouse --no-headers | grep -v 'Terminating' | awk '{print $1}' | head -n1)"

kubectl logs "${POD}" -n "$BOOKWAREHOUSE_NAMESPACE" -c bookwarehouse --tail=100 -f
