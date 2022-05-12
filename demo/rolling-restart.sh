#!/bin/bash



# This script performs a rolling restart of the deployments listed below.
# This is part of the OSM Bookstore demo helper scripts.



set -aueo pipefail

# shellcheck disable=SC1091
source .env

BOOKBUYER_NAMESPACE="${BOOKBUYER_NAMESPACE:-bookbuyer}"
BOOKSTORE_NAMESPACE="${BOOKSTORE_NAMESPACE:-bookstore}"
BOOKTHIEF_NAMESPACE="${BOOKTHIEF_NAMESPACE:-bookthief}"
BOOKWAREHOUSE_NAMESPACE="${BOOKWAREHOUSE_NAMESPACE:-bookwarehouse}"



kubectl rollout restart deployment bookbuyer      -n "$BOOKBUYER_NAMESPACE"
kubectl rollout restart deployment bookstore-v1   -n "$BOOKSTORE_NAMESPACE"
kubectl rollout restart deployment bookstore-v2   -n "$BOOKSTORE_NAMESPACE"
kubectl rollout restart deployment bookthief      -n "$BOOKTHIEF_NAMESPACE"
kubectl rollout restart deployment bookwarehouse  -n "$BOOKWAREHOUSE_NAMESPACE"
