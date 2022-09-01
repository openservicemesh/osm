#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env
DEPLOY_ON_OPENSHIFT="${DEPLOY_ON_OPENSHIFT:-false}"
USE_PRIVATE_REGISTRY="${USE_PRIVATE_REGISTRY:-false}"
MESH_NAME="${MESH_NAME:-osm}"

kubectl create ns zookeeper 

bin/osm namespace add --mesh-name "$MESH_NAME" zookeeper

bin/osm metrics enable --namespace zookeeper


helm install kafka bitnami/zookeeper --set replicaCount=3 --set serviceAccount.create=true --set serviceAccount.name=zookeeper --namespace zookeeper

if [ "$DEPLOY_ON_OPENSHIFT" = true ] ; then
    oc adm policy add-scc-to-user privileged -z "zookeeper" -n "zookeeper"
    if [ "$USE_PRIVATE_REGISTRY" = true ]; then
        oc secrets link "zookeeper" "$CTR_REGISTRY_CREDS_NAME" --for=pull -n "zookeeper"
    fi
fi

kubectl apply -f - <<EOF
apiVersion: specs.smi-spec.io/v1alpha4
kind: TCPRoute
metadata:
  name: zookeeper
  namespace: zookeeper
spec:
  matches:
    ports:
    - 2181
    - 3181
---
apiVersion: specs.smi-spec.io/v1alpha4
kind: TCPRoute
metadata:
  name: zookeeper-internal
  namespace: zookeeper
spec:
  matches:
    ports:
    - 2181
    - 3181
    - 2888
    - 3888
---
kind: TrafficTarget
apiVersion: access.smi-spec.io/v1alpha3
metadata:
  name: zookeeper
  namespace: zookeeper
spec:
  destination:
    kind: ServiceAccount
    name: zookeeper
    namespace: zookeeper
  rules:
  - kind: TCPRoute
    name: zookeeper
  sources:
  - kind: ServiceAccount
    name: kafka
    namespace: kafka
---
kind: TrafficTarget
apiVersion: access.smi-spec.io/v1alpha3
metadata:
  name: zookeeper-internal
  namespace: zookeeper
spec:
  destination:
    kind: ServiceAccount
    name: zookeeper
    namespace: zookeeper
  rules:
  - kind: TCPRoute
    name: zookeeper-internal
  sources:
  - kind: ServiceAccount
    name: zookeeper
    namespace: zookeeper
EOF

