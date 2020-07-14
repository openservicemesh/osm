#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

echo -e "Deploy bookstore-vm demo service"
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: bookstore-vm
  namespace: $K8S_NAMESPACE
  labels:
    app: bookstore-vm
spec:
  ports:
  - port: 80
    targetPort: 80
    name: app-port
  selector:
    app: bookstore-vm
---

apiVersion: osm.osm.k8s.io/v1
kind: AzureResource
metadata:
  name: bookstore
  namespace: $K8S_NAMESPACE
  labels:
    app: bookstore-vm
spec:
  resourceid: /subscriptions/your-subscription-id/resourceGroups/your-resource-group-name/providers/Microsoft.Compute/virtualMachines/vm-name

EOF
