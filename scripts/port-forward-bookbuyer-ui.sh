#!/bin/bash


# This script port forwards from the BOOKBUYER pod to local port 8080


# shellcheck disable=SC1091
source .env

BOOKBUYER_NAMESPACE="${BOOKBUYER_NAMESPACE:-bookbuyer}"

POD="$(kubectl get pods --selector app=bookbuyer,version=v1 -n "$BOOKBUYER_NAMESPACE" --no-headers | grep 'Running' | awk '{print $1}')"

kubectl port-forward "$POD" -n "$BOOKBUYER_NAMESPACE" 8080:80
