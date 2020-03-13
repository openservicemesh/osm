#!/bin/bash

# shellcheck disable=SC1091
source .env

NS="${K8S_NAMESPACE}-bookstore"

POD="$(kubectl get pods -n "$NS" --show-labels --selector app=bookstore-1 --no-headers | grep -v 'Terminating' | awk '{print $1}' | head -n1)"

kubectl logs "$POD" -n "$NS" -c bookstore-1 --tail=100
