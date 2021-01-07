#!/bin/bash



# This script joins the list of namespaces to the existing service mesh.
# This is a helper script part of brownfield OSM demo.



set -aueo pipefail

# shellcheck disable=SC1091
source .env


./bin/osm namespace add "${BOOKBUYER_NAMESPACE:-bookbuyer}"         --mesh-name "${MESH_NAME:-osm}"
./bin/osm namespace add "${BOOKSTORE_NAMESPACE:-bookstore}"         --mesh-name "${MESH_NAME:-osm}"
./bin/osm namespace add "${BOOKTHIEF_NAMESPACE:-bookthief}"         --mesh-name "${MESH_NAME:-osm}"
./bin/osm namespace add "${BOOKWAREHOUSE_NAMESPACE:-bookwarehouse}" --mesh-name "${MESH_NAME:-osm}"


kubectl patch ConfigMap \
        -n "${K8S_NAMESPACE}" osm-config \
        --type merge \
        --patch '{"data":{"permissive_traffic_policy_mode":"false"}}'


# Create a top level service
echo -e "Deploy bookstore Service"
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  labels:
    app: bookstore
  name: bookstore
  namespace: bookstore
spec:
  ports:
  - name: bookstore-port
    port: 80
    protocol: TCP
    targetPort: 80
  selector:
    app: bookstore
EOF

sleep 3


./demo/rolling-restart.sh
