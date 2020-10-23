#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

bin/osm mesh uninstall -f --mesh-name "$MESH_NAME" --osm-namespace "$K8S_NAMESPACE"

for ns in "$BOOKWAREHOUSE_NAMESPACE" "$BOOKBUYER_NAMESPACE" "$BOOKSTORE_NAMESPACE" "$BOOKTHIEF_NAMESPACE" "$K8S_NAMESPACE"; do
    kubectl delete namespace "$ns" --ignore-not-found --wait &
done

# Clean up Hashicorp Vault deployment
kubectl delete deployment vault -n "$K8S_NAMESPACE" --ignore-not-found --wait &
kubectl delete service vault -n "$K8S_NAMESPACE" --ignore-not-found --wait &

wait
