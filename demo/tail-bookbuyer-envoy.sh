#!/bin/bash

# shellcheck disable=SC1091
source .env

NS="${K8S_NAMESPACE}-bookbuyer"

kubectl describe pod "$(kubectl get pods -n "$NS" --show-labels --selector app=client --no-headers | grep -v 'Terminating' | awk '{print $1}' | head -n1)" -n "$NS"

POD="$(kubectl get pods -n "$NS" --show-labels --selector app=bookbuyer --no-headers | grep -v 'Terminating' | awk '{print $1}' | head -n1)"

kubectl logs "${POD}" -n "$NS" -c envoyproxy --tail=100 -f
