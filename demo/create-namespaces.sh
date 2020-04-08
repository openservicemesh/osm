#!/bin/bash

set -auexo pipefail

source .env

# The demo uses the following namespaces defined by environment variables:
# 1. K8S_NAMESPACE: OSM's namespace
# 2. BOOKBUYER_NAMESPACE: Namespace for the Bookbuyer service
# 3. BOOKSTORE_NAMESPACE: Namespace for the Bookstore service
kubectl create namespace "$K8S_NAMESPACE"
for ns in "$BOOKBUYER_NAMESPACE" "$BOOKSTORE_NAMESPACE" "$BOOKTHIEF_NAMESPACE"; do
    kubectl create namespace "$ns"
    kubectl label  namespaces "$ns" openservicemesh.io/monitor="$OSM_ID"
done
