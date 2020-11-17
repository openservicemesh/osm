#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

for ns in "$BOOKWAREHOUSE_NAMESPACE" "$BOOKBUYER_NAMESPACE" "$BOOKSTORE_NAMESPACE" "$BOOKTHIEF_NAMESPACE"; do
    kubectl create namespace "$ns" --save-config
    ./scripts/create-container-registry-creds.sh "$ns"
done

# Add namespaces to the mesh
bin/osm namespace add --mesh-name "$MESH_NAME" "$BOOKWAREHOUSE_NAMESPACE" "$BOOKBUYER_NAMESPACE" "$BOOKSTORE_NAMESPACE" "$BOOKTHIEF_NAMESPACE"

# Enable metrics for pods belonging to app namespaces
bin/osm metrics enable --namespace "$BOOKWAREHOUSE_NAMESPACE, $BOOKBUYER_NAMESPACE, $BOOKSTORE_NAMESPACE, $BOOKTHIEF_NAMESPACE"
