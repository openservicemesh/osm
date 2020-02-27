#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

SVC="bookstore-vm"

echo -e "Deploy $SVC demo service"
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: $SVC
  namespace: $K8S_NAMESPACE
  labels:
    app: $SVC
spec:
  ports:
  - port: 80
    targetPort: 80
    name: app-port
  selector:
    app: $SVC
  type: NodePort
---

apiVersion: smc.osm.k8s.io/v1
kind: AzureResource
metadata:
  name: bookstore
  namespace: $K8S_NAMESPACE
  labels:
    app: $SVC
spec:
  resourceid: /subscriptions/your-subscription-id/resourceGroups/your-resource-group-name/providers/Microsoft.Compute/virtualMachines/vm-name
EOF
