#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env
CURL_NAMESPACE="${CURL_NAMESPACE:-curl}"
HELLO_NAMESPACE="${HELLO_NAMESPACE:-hello}"
SHOULD_INSTALL_OSM_FOR_CURL_HELLO_DEMO="${SHOULD_INSTALL_OSM_FOR_CURL_HELLO_DEMO:-true}"

for ns in "$CURL_NAMESPACE" "$HELLO_NAMESPACE"; do
    kubectl create namespace "$ns" --save-config
done

if [ "$SHOULD_INSTALL_OSM_FOR_CURL_HELLO_DEMO" = true ] ; then
  # Add namespaces to the mesh
  bin/osm namespace add --mesh-name "$MESH_NAME" "$CURL_NAMESPACE" "$HELLO_NAMESPACE"

  # Enable metrics for pods belonging to app namespaces
  bin/osm metrics enable --namespace "$CURL_NAMESPACE, $HELLO_NAMESPACE"
fi
