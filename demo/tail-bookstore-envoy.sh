#!/bin/bash

# shellcheck disable=SC1091
source .env

POD="$(kubectl get pods -n "$K8S_NAMESPACE" --show-labels --selector app=bookstore-1 --no-headers | grep -v 'Terminating' | awk '{print $1}' | head -n1)"

kubectl logs "$POD" -n "$K8S_NAMESPACE" -c envoyproxy --tail=100 -f
