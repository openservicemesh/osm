#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

for ns in "$BOOKWAREHOUSE_NAMESPACE" "$BOOKBUYER_NAMESPACE" "$BOOKSTORE_NAMESPACE" "$BOOKTHIEF_NAMESPACE"; do
    kubectl label namespaces "$ns" openservicemesh.io/monitor="$K8S_NAMESPACE" --overwrite="true"
    kubectl label namespaces "$ns" openservicemesh.io/monitored-by=osm --overwrite="true"
done


kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap

metadata:
  name: osm-config
  namespace: $K8S_NAMESPACE

data:
  allow_all: true

EOF

sleep 3

./demo/rolling-restart.sh
