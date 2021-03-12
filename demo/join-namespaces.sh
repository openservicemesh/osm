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

./bin/osm mesh upgrade --osm-namespace "${K8S_NAMESPACE}" --mesh-name "${MESH_NAME:-osm}" --container-registry "${CTR_REGISTRY}" --osm-image-tag "${CTR_TAG}" --enable-permissive-traffic-policy=false


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
    port: 14001
    protocol: TCP
    targetPort: 14001
  selector:
    app: bookstore
EOF

sleep 3


./demo/rolling-restart.sh
