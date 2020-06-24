#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

kubectl delete crds osmconfigs.osm.k8s.io || true

kubectl apply -f ./crd/OSMConfig.yaml || true

cat <<EOF | kubectl apply -f -
apiVersion: osm.k8s.io/v1
kind: OSMConfig
metadata:
  name: osm-config
  namespace: $K8S_NAMESPACE
spec:
  logVerbosity: trace
  namespaces:
    - "$BOOKWAREHOUSE_NAMESPACE"
    - "$BOOKBUYER_NAMESPACE"
    - "$BOOKSTORE_NAMESPACE"
    - "$BOOKTHIEF_NAMESPACE"
  ingresses:
    - "something"
EOF


cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: osm-config
  namespace: $K8S_NAMESPACE
data:
  osm.conf: |
        logVerbosity: trace
        namespaces:
          - "$BOOKWAREHOUSE_NAMESPACE"
          - "$BOOKBUYER_NAMESPACE"
          - "$BOOKSTORE_NAMESPACE"
          - "$BOOKTHIEF_NAMESPACE"
        ingresses:
          - "something"
EOF
