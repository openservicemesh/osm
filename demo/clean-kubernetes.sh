#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

MESH_NAME="${MESH_NAME:-osm}"
K8S_NAMESPACE="${K8S_NAMESPACE:-osm-system}"
BOOKBUYER_NAMESPACE="${BOOKBUYER_NAMESPACE:-bookbuyer}"
BOOKSTORE_NAMESPACE="${BOOKSTORE_NAMESPACE:-bookstore}"
BOOKTHIEF_NAMESPACE="${BOOKTHIEF_NAMESPACE:-bookthief}"
BOOKWAREHOUSE_NAMESPACE="${BOOKWAREHOUSE_NAMESPACE:-bookwarehouse}"
TIMEOUT="${TIMEOUT:-90s}"

bin/osm uninstall -f --mesh-name "$MESH_NAME" --osm-namespace "$K8S_NAMESPACE"

for ns in "$BOOKWAREHOUSE_NAMESPACE" "$BOOKBUYER_NAMESPACE" "$BOOKSTORE_NAMESPACE" "$BOOKTHIEF_NAMESPACE" "$K8S_NAMESPACE"; do
    kubectl delete namespace "$ns" --ignore-not-found --wait --timeout="$TIMEOUT" &
done

# Clean up Hashicorp Vault deployment
kubectl delete deployment vault -n "$K8S_NAMESPACE" --ignore-not-found --wait --timeout="$TIMEOUT" &
kubectl delete service vault -n "$K8S_NAMESPACE" --ignore-not-found --wait --timeout="$TIMEOUT" &

wait
