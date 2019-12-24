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
  name: bookstore.azuremesh
  namespace: "$K8S_NAMESPACE"
spec:
  service: bookstore.azuremesh
  backends:

  - service: bookstore-1
    weight: 100

  - service: bookstore-2
    weight: 5

  - service: "/subscriptions/$AZURE_SUBSCRIPTION/resourceGroups/$AZURE_RESOURCE_GROUP/providers/Microsoft.Compute/virtualMachines/$VM_NAME"
    weight: 100
EOF
