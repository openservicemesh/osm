#!/bin/bash

# shellcheck disable=SC1091
source .env

kubectl describe pod "$(kubectl get pods -n "$K8S_NAMESPACE" --show-labels --selector app=client --no-headers | grep -v 'Terminating' | awk '{print $1}' | head -n1)" -n "$K8S_NAMESPACE"

POD="$(kubectl get pods -n "$K8S_NAMESPACE" --show-labels --selector app=bookthief --no-headers | grep -v 'Terminating' | awk '{print $1}' | head -n1)"

kubectl logs "${POD}" -n "$K8S_NAMESPACE" -c envoyproxy --tail=100 -f
