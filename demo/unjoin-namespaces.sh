#!/bin/bash



# This script removes the list of namespaces from the OSM.
# This is a helper script part of the OSM Brownfield Deployment Demo.



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

    # The trailing minus (-) sign after MESH_NAME below is deliberate - this REMOVES the label
    kubectl label namespaces "$ns" openservicemesh.io/monitored-by="$MESH_NAME"- || true

done


kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap

metadata:
  name: osm-config
  namespace: $K8S_NAMESPACE

data:

    permissive_traffic_policy_mode: true

EOF


# Create a top level service
echo -e "Deploy bookstore Service"
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service

metadata:
  name: bookstore
  namespace: $BOOKSTORE_NAMESPACE

spec:
  ports:
  - port: 80
    name: bookstore-port

  selector:
    app: bookstore-v1

EOF


./demo/rolling-restart.sh
