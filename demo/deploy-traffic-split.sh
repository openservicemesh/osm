#!/bin/bash

set -aueo pipefail

source .env

kubectl apply -f https://raw.githubusercontent.com/deislabs/smi-sdk-go/master/crds/split.yaml

kubectl create namespace "$K8S_NAMESPACE" || true

VM_NAME="myVM"

kubectl apply -f - <<EOF
apiVersion: split.smi-spec.io/v1alpha2
kind: TrafficSplit
metadata:
  name: bookstore.mesh
  namespace: "$K8S_NAMESPACE"
spec:
  service: bookstore.mesh
  backends:

  - service: bookstore-1
    weight: 80

  - service: bookstore-2
    weight: 15

  - service: "/subscriptions/$AZURE_SUBSCRIPTION/resourceGroups/$AZURE_RESOURCE_GROUP/providers/Microsoft.Compute/virtualMachines/$VM_NAME"
    weight: 5
EOF
