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


./bin/osm namespace remove "${BOOKWAREHOUSE_NAMESPACE:-bookbuyer}" --mesh-name "${MESH_NAME:-osm}"
./bin/osm namespace remove "${BOOKBUYER_NAMESPACE:-bookbuyer}"     --mesh-name "${MESH_NAME:-osm}"
./bin/osm namespace remove "${BOOKSTORE_NAMESPACE:-bookbuyer}"     --mesh-name "${MESH_NAME:-osm}"
./bin/osm namespace remove "${BOOKTHIEF_NAMESPACE:-bookbuyer}"     --mesh-name "${MESH_NAME:-osm}"


kubectl patch ConfigMap \
        -n "${K8S_NAMESPACE}" osm-config \
        --type merge \
        --patch '{"data":{"permissive_traffic_policy_mode":"true"}}'


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
