#!/bin/bash

# This is script is only meant to clean up leaked resources.
# Please use osm cli or helm uninstall if possible.

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
    --output-dir ./out \

kubectl delete -f out/osm/templates
