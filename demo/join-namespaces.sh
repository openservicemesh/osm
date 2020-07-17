#!/bin/bash



# This script joins the list of namespaces to the existing service mesh.
# This is a helper script part of brownfield OSM demo.



set -aueo pipefail

# shellcheck disable=SC1091
source .env



K8S_NAMESPACE="${K8S_NAMESPACE:-osm-system}"
BOOKBUYER_NAMESPACE="${BOOKBUYER_NAMESPACE:-bookbuyer}"
BOOKSTORE_NAMESPACE="${BOOKSTORE_NAMESPACE:-bookstore}"
BOOKTHIEF_NAMESPACE="${BOOKTHIEF_NAMESPACE:-bookthief}"
BOOKWAREHOUSE_NAMESPACE="${BOOKWAREHOUSE_NAMESPACE:-bookwarehouse}"

MESH_NAME="${MESH_NAME:-osm}"


for ns in "$BOOKWAREHOUSE_NAMESPACE" "$BOOKBUYER_NAMESPACE" "$BOOKSTORE_NAMESPACE" "$BOOKTHIEF_NAMESPACE"; do
    kubectl label namespaces "$ns" openservicemesh.io/monitored-by="$MESH_NAME" --overwrite="true"
done


kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap

metadata:
  name: osm-config
  namespace: $K8S_NAMESPACE

data:
  permissive_traffic_policy_mode: "true"

EOF

sleep 3

./demo/rolling-restart.sh
