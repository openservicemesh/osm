#!/bin/bash

# shellcheck disable=SC1091
source .env

selector="$1"
thisScript="$(dirname "$0")/$(basename "$0")"

if [ -z "$selector" ]; then
    echo "Usage: $thisScript <selector>"
    exit 1
fi

POD="$(kubectl get pods -n "$BOOKSTORE_NAMESPACE" --show-labels --selector "$selector" --no-headers | grep -v 'Terminating' | awk '{print $1}' | head -n1)"

kubectl logs "$POD" -n "$BOOKSTORE_NAMESPACE" -c bookstore --tail=100
