#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

kubectl apply -f https://raw.githubusercontent.com/deislabs/smi-sdk-go/v0.2.0/crds/access.yaml

NAME="bookstore"
NS="${K8S_NAMESPACE}-${NAME}"

kubectl apply -f - <<EOF
kind: TrafficTarget
apiVersion: access.smi-spec.io/v1alpha1
metadata:
  name: bookstore-service-new
  namespace: "$NS"
destination:
  kind: ServiceAccount
  name: bookstore-2-serviceaccount
  namespace: "$NS"
specs:
- kind: HTTPRouteGroup
  name: bookstore-service-routes
  matches:
  - incrementcounter
sources:
- kind: ServiceAccount
  name: bookbuyer-serviceaccount
  namespace: "$NS"
EOF
