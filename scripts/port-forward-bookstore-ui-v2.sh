#!/bin/bash


# This script forwards BOOKSTORE port 14001 to local host port 8082


# shellcheck disable=SC1091
source .env

BOOKSTOREv2_LOCAL_PORT="${BOOKSTOREv2_LOCAL_PORT:-8082}"
POD="$(kubectl get pods --selector app=bookstore,version=v2 -n "$BOOKSTORE_NAMESPACE" --no-headers | grep 'Running' | awk 'NR==1{print $1}')"

kubectl port-forward "$POD" -n "$BOOKSTORE_NAMESPACE" "$BOOKSTOREv2_LOCAL_PORT":14001
