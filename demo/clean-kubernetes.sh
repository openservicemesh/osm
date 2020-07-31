#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

for ns in "$BOOKWAREHOUSE_NAMESPACE" "$BOOKBUYER_NAMESPACE" "$BOOKSTORE_NAMESPACE" "$BOOKTHIEF_NAMESPACE" "$K8S_NAMESPACE"; do
    kubectl delete namespace "$ns" --ignore-not-found
done

# Clean up Hashicorp Vault deployment
kubectl delete deployment vault -n "$K8S_NAMESPACE" --ignore-not-found
kubectl delete service vault -n "$K8S_NAMESPACE" --ignore-not-found
