#!/bin/bash

# This script is only for cleaning up leaked resources.
# Please use osm cli or helm uninstall if possible.

# shellcheck disable=SC1091
source .env

MESH_NAME="${MESH_NAME:-osm}"
K8S_NAMESPACE="${K8S_NAMESPACE:-osm-system}"
DEPLOY_GRAFANA="${DEPLOY_GRAFANA:-true}"
ENABLE_FLUENTBIT="${ENABLE_FLUENTBIT:-false}"
DEPLOY_PROMETHEUS="${DEPLOY_PROMETHEUS:-true}"


helm template "$MESH_NAME" ./charts/osm \
    --set OpenServiceMesh.osmNamespace="$K8S_NAMESPACE" \
    --set OpenServiceMesh.deployGrafana="$DEPLOY_GRAFANA" \
    --set OpenServiceMesh.enableFluentbit="$ENABLE_FLUENTBIT" \
    --set OpenServiceMesh.deployPrometheus="$DEPLOY_PROMETHEUS" \
    | kubectl delete --ignore-not-found -f -
