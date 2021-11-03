#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

CURL_NAMESPACE="${CURL_NAMESPACE:-curl}"
HELLO_NAMESPACE="${HELLO_NAMESPACE:-hello}"
TIMEOUT="${TIMEOUT:-90s}"

echo "Uninstalling OSM (if present) from previous runs"
bin/osm uninstall mesh -f --mesh-name "$MESH_NAME" --osm-namespace "$K8S_NAMESPACE"

echo "Uninstalling OSM cluster-wide-resources (if present) from previous runs"
bin/osm uninstall cluster-wide-resources -f --mesh-name "$MESH_NAME" --osm-namespace "$K8S_NAMESPACE" --ca-bundle-secret-name "$CA_BUNDLE_SECRET_NAME"

for ns in "$BOOKWAREHOUSE_NAMESPACE" "$BOOKBUYER_NAMESPACE" "$BOOKSTORE_NAMESPACE" "$BOOKTHIEF_NAMESPACE" "$K8S_NAMESPACE"; do
    kubectl delete namespace "$ns" --ignore-not-found --wait --timeout="$TIMEOUT" &
done

# These namespaces are used for the demo where we benchmark request performance from curlimages/curl to tutum/hello-world
for ns in "$CURL_NAMESPACE" "$HELLO_NAMESPACE"; do
    kubectl delete namespace "$ns" --ignore-not-found --wait --timeout="$TIMEOUT" &
done

# Clean up Hashicorp Vault deployment
kubectl delete deployment vault -n "$K8S_NAMESPACE" --ignore-not-found --wait --timeout="$TIMEOUT" &
kubectl delete service vault -n "$K8S_NAMESPACE" --ignore-not-found --wait --timeout="$TIMEOUT" &

wait
