#!/bin/bash

# shellcheck disable=SC1091
source .env

backend="$1"
thisScript="$(dirname "$0")/$(basename "$0")"

if [ -z "$backend" ]; then
    echo "Usage: $thisScript <backend-name>"
    exit 1
fi

POD="$(kubectl get pods -n "$BOOKSTORE_NAMESPACE" --show-labels --selector app="$backend" --no-headers | grep -v 'Terminating' | awk '{print $1}' | head -n1)"

kubectl logs "$POD" -n "$BOOKSTORE_NAMESPACE" -c envoy --tail=100 -f
