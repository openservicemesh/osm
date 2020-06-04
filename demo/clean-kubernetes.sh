#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

WEBHOOK_NAME="osm-webhook-$K8S_NAMESPACE"

kubectl delete mutatingwebhookconfiguration "$WEBHOOK_NAME" --ignore-not-found=true
for ns in "$BOOKWAREHOUSE_NAMESPACE" "$BOOKBUYER_NAMESPACE" "$BOOKSTORE_NAMESPACE" "$BOOKTHIEF_NAMESPACE" "$K8S_NAMESPACE"; do
    kubectl delete namespace "$ns" || true
done
kubectl delete clusterrole ads || true
kubectl delete clusterrolebinding ads || true

# cleaning all prometheus related resources
kubectl delete clusterrole "$PROMETHEUS_SVC" || true
kubectl delete clusterrolebinding "$PROMETHEUS_SVC" || true

# Clean up Hashicorp Vault deployment
kubectl delete deployment vault -n "$K8S_NAMESPACE" || true
kubectl delete service vault -n "$K8S_NAMESPACE" || true
