#!/bin/bash

# shellcheck disable=SC1091

BOOKSTOREv1_LOCAL_PORT="${BOOKSTOREv1_LOCAL_PORT:-8081}"
POD="$(kubectl get pods --selector app=bookstore --selector version=v1 -n "$BOOKSTORE_NAMESPACE" --no-headers | grep 'Running' | awk '{print $1}')"

kubectl port-forward "$POD" -n "$BOOKSTORE_NAMESPACE" 15001:15000
