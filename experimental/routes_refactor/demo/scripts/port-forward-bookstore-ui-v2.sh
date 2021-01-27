#!/bin/bash


# This script forwards port 80 from the BOOKSTORE to local port 8081.


# shellcheck disable=SC1091

BOOKSTOREv1_LOCAL_PORT="${BOOKSTOREv1_LOCAL_PORT:-8082}"
POD="$(kubectl get pods --selector app=bookstore --selector version=v2 -n "$BOOKSTORE_NAMESPACE" --no-headers | grep 'Running' | awk '{print $1}')"

kubectl port-forward "$POD" -n "$BOOKSTORE_NAMESPACE" "$BOOKSTOREv1_LOCAL_PORT":80