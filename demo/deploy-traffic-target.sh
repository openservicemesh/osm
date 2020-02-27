#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

kubectl apply -f https://raw.githubusercontent.com/deislabs/smi-sdk-go/master/crds/access.yaml

kubectl apply -f - <<EOF
kind: TrafficTarget
apiVersion: access.smi-spec.io/v1alpha1
metadata:
  name: bookstore-service
  namespace: "$K8S_NAMESPACE"
destination:
  # (todo): use service account
  kind: ServiceAccount
  name: bookstore-1-serviceaccount
  namespace: "$K8S_NAMESPACE"
specs:
- kind: HTTPRouteGroup
  name: bookstore-service-routes
  matches:
  - counter
sources:
# (todo): use service account
- kind: ServiceAccount
  name: bookbuyer-serviceaccount
  namespace: "$K8S_NAMESPACE"
EOF
