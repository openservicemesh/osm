#!/bin/bash

# shellcheck disable=SC1091
source .env

POD="$(kubectl get pods -n "$BOOKSTORE_NAMESPACE" --show-labels --selector app=bookstore-1 --no-headers | grep -v 'Terminating' | awk '{print $1}' | head -n1)"

kubectl logs "$POD" -n "$BOOKSTORE_NAMESPACE" -c envoyproxy --tail=100 -f
