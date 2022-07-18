#!/bin/bash


# This script forwards port 14001 from the BOOKSTORE to local port 8081.


# shellcheck disable=SC1091
source .env

BOOKSTOREv1_LOCAL_PORT="${BOOKSTOREv1_LOCAL_PORT:-8081}"
POD="$(kubectl get pods --selector app=bookstore,version=v1 -n "$BOOKSTORE_NAMESPACE" --no-headers | grep 'Running' | awk 'NR==1{print $1}')"

kubectl port-forward "$POD" -n "$BOOKSTORE_NAMESPACE" "$BOOKSTOREv1_LOCAL_PORT":14001
