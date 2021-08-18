#!/bin/bash

set -auexo pipefail

# shellcheck disable=SC1091
source .env

VERSION=${1:-v1}
SVC="bookstore-$VERSION"
ALPHA_CLUSTER="${ALPHA_CLUSTER:-alpha}"
BETA_CLUSTER="${BETA_CLUSTER:-beta}"
BOOKSTORE_NAMESPACE="${BOOKSTORE_NAMESPACE:-bookstore}"

kubectl config use-context "$BETA_CLUSTER"
BETA_OSM_GATEWAY_IP=$(kubectl get svc -n 'osm-system' --selector app=osm-multicluster-gateway -o json | jq -r '.items[0].status.loadBalancer.ingress[0].ip')


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
    weight: 20
  serviceAccount: bookstore-v1
EOF
