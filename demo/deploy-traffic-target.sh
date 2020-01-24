#!/bin/bash

set -aueo pipefail

source .env

kubectl apply -f https://raw.githubusercontent.com/deislabs/smi-sdk-go/master/crds/access.yaml

kubectl create namespace "$K8S_NAMESPACE" || true

VM_NAME="myVM"

kubectl apply -f - <<EOF
kind: TrafficTarget
apiVersion: access.smi-spec.io/v1alpha1
metadata:
  name: bookstore-service-counter
  namespace: "$K8S_NAMESPACE"
destination:
  # (todo): use service account
  kind: Service
  name: bookstore
  namespace: "$K8S_NAMESPACE"
specs:
- kind: HTTPRouteGroup
  name: bookstore-service-routes
  matches:
  - counter
sources:
# (todo): use service account
- kind: Service
  name: bookbuyer
  namespace: "$K8S_NAMESPACE"
EOF