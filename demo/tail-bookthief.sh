#!/bin/bash

# shellcheck disable=SC1091
source .env

POD="$(kubectl get pods -n "$BOOKTHIEF_NAMESPACE" --show-labels --selector app=bookthief --no-headers | grep -v 'Terminating' | awk '{print $1}' | head -n1)"

kubectl logs "${POD}" -n "$BOOKTHIEF_NAMESPACE" -c bookthief --tail=100 -f