#!/bin/bash

set -auexo pipefail

# shellcheck disable=SC1091
source .env

VERSION=${1:-v1}
SVC="bookstore-$VERSION"
ALPHA_CLUSTER="${ALPHA_CLUSTER:-alpha}"
BETA_CLUSTER="${BETA_CLUSTER:-beta}"
BOOKSTORE_NAMESPACE="${BOOKSTORE_NAMESPACE:-bookstore}"
K8S_NAMESPACE="${K8S_NAMESPACE:-osm-system}"

kubectl config use-context "$BETA_CLUSTER"
# TODO : the Pod IP of osm-multicluster-gateway is used cause the clusters are in the same vnet, this needs to be updated to leverage the IP of osm-multicluster-gateway service
BETA_OSM_GATEWAY_IP=$(kubectl get pods -n "$K8S_NAMESPACE" --selector app=osm-multicluster-gateway -o json | jq -r '.items[].status.podIP')


kubectl config use-context "$ALPHA_CLUSTER"


kubectl apply -f - <<EOF
apiVersion: config.openservicemesh.io/v1alpha1
kind: MultiClusterService
metadata:
  namespace: $BOOKSTORE_NAMESPACE
  name: $SVC
spec:
  clusters:
  - name: $BETA_CLUSTER
    address: $BETA_OSM_GATEWAY_IP:15443
  serviceAccount: bookstore-v1
EOF
