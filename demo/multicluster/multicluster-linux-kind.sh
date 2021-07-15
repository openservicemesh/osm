#!/bin/bash

# This script should be executed after demo/run-osm-multicluster-demo.sh

set -auexo pipefail

# shellcheck disable=SC1091
source .env

echo "Please ensure contexts specified by MULTICLUSTER_CONTEXTS in .env are Kind clusters."

# shellcheck disable=SC2207
CONTEXTS_ARRAY=($(echo "$MULTICLUSTER_CONTEXTS" | tr ';' ' '))
CLIENT_CONTEXT=${CONTEXTS_ARRAY[0]}
SERVER_CONTEXT=${CONTEXTS_ARRAY[1]}

echo Client context is "${CLIENT_CONTEXT}"
echo Server context is "${SERVER_CONTEXT}"

GATEWAY_PORT=14080

# === Configuration on server cluster
kubectl config use-context "$SERVER_CONTEXT"

# install metallb namespace
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/master/manifests/namespace.yaml
# memberlist secrets
# kubectl create secret generic -n metallb-system memberlist --from-literal=secretkey="$(openssl rand -base64 128)"
# apply metallb manifest
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/master/manifests/metallb.yaml
# wait for metallb to come up
sleep 5
echo MetalLB IP address is "$(docker network inspect -f '{{.IPAM.Config}}' kind)"
echo Please input Kubernetes services external IP range, e.g. 172.18.255.200-172.18.255.250
read -r IP_RANGE
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  namespace: metallb-system
  name: config
data:
  config: |
    address-pools:
    - name: default
      protocol: layer2
      addresses:
      - ${IP_RANGE}
EOF

# create osm-gateway LoadBalancer
kubectl apply -f - <<EOF
kind: Service
apiVersion: v1
metadata:
  name: osm-gateway
  namespace: osm-system
spec:
  type: LoadBalancer
  selector:
    app: osm-gateway
  ports:
  - port: $GATEWAY_PORT
EOF

GATEWAY_EXT_IP=$(kubectl get svc osm-gateway -n osm-system -o jsonpath="{.status.loadBalancer.ingress[0].ip}")

# === Configure MulticlusterService on client cluster
kubectl config use-context "$CLIENT_CONTEXT"
kubectl patch MulticlusterService \
        --namespace bookstore \
        bookstore \
        --type merge \
        --patch '{"spec":{"clusters":[{"name":"beta","address":"'"${GATEWAY_EXT_IP}"':'"${GATEWAY_PORT}"'"}]}}'