#!/bin/bash


# This script forwards BOOKSTORE port 80 to local host port 8082


# shellcheck disable=SC1091
source .env

POD="$(kubectl get pods --selector app=bookstore,version=v2 -n "$BOOKSTORE_NAMESPACE" --no-headers | grep 'Running' | awk '{print $1}')"

kubectl port-forward "$POD" -n "$BOOKSTORE_NAMESPACE" 8082:80
