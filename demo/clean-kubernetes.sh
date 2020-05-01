#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

WEBHOOK_NAME="$K8S_NAMESPACE-ads-webhook"

kubectl delete mutatingwebhookconfiguration "$WEBHOOK_NAME" --ignore-not-found=true
for ns in "$BOOKBUYER_NAMESPACE" "$BOOKSTORE_NAMESPACE" "$BOOKTHIEF_NAMESPACE" "$K8S_NAMESPACE"; do
    kubectl delete namespace "$ns" || true
done
kubectl delete clusterrole osm-xds || true
kubectl delete clusterrolebinding osm-xds || true

# cleaning all prometheus related resources
kubectl delete clusterrole "$PROMETHEUS_SVC" || true
kubectl delete clusterrolebinding "$PROMETHEUS_SVC" || true

# Clean up Hashicorp Vault deployment
kubectl delete deployment vault -n "$K8S_NAMESPACE" || true
kubectl delete service vault -n "$K8S_NAMESPACE" || true
