#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

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
    - $BOOKWAREHOUSE_NAMESPACE
    - $BOOKBUYER_NAMESPACE
    - $BOOKSTORE_NAMESPACE
    - $BOOKTHIEF_NAMESPACE
  ingresses:
    - something
EOF
