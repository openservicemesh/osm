#!/bin/bash

# shellcheck disable=SC1091
source .env

BOOKSTOREv1_LOCAL_PORT="${BOOKSTOREv1_LOCAL_PORT:-8082}"
POD="$(kubectl get pods --selector app=bookstore --selector version=v2 -n "$BOOKSTORE_NAMESPACE" --no-headers | grep 'Running' |awk 'NR==1{print $1}')"

kubectl port-forward "$POD" -n "$BOOKSTORE_NAMESPACE" 15002:15000
