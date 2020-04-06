#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

kubectl apply -f https://raw.githubusercontent.com/deislabs/smi-sdk-go/v0.2.0/crds/access.yaml

kubectl apply -f - <<EOF
kind: TrafficTarget
apiVersion: access.smi-spec.io/v1alpha1
metadata:
  name: bookbuyer-access-bookstore-1
  namespace: "$BOOKSTORE_NAMESPACE"
destination:
  kind: ServiceAccount
  name: bookstore-1-serviceaccount
  namespace: "$BOOKSTORE_NAMESPACE"
specs:
- kind: HTTPRouteGroup
  name: bookstore-service-routes
  matches:
  - books-bought
sources:
- kind: ServiceAccount
  name: bookbuyer-serviceaccount
  namespace: "$BOOKBUYER_NAMESPACE"

---

kind: TrafficTarget
apiVersion: access.smi-spec.io/v1alpha1
metadata:
  name: bookbuyer-access-bookstore-2
  namespace: "$BOOKSTORE_NAMESPACE"
destination:
  kind: ServiceAccount
  name: bookstore-2-serviceaccount
  namespace: "$BOOKSTORE_NAMESPACE"
specs:
- kind: HTTPRouteGroup
  name: bookstore-service-routes
  matches:
  - buy-a-book
sources:
- kind: ServiceAccount
  name: bookbuyer-serviceaccount
  namespace: "$BOOKBUYER_NAMESPACE"
EOF

# To remove this, annotate the POD and also update the test to not
# expect a 404. This is because if an SMI policy is not defined but sidecar
# is injected, the gRPC xDS stream will be dropped by ADS because there is
# no service/service-account associated with the service.
kubectl apply -f - <<EOF
kind: TrafficTarget
apiVersion: access.smi-spec.io/v1alpha1
metadata:
  name: bookthief-access-bookstore-1
  namespace: "$BOOKSTORE_NAMESPACE"
destination:
  kind: ServiceAccount
  name: bookstore-1-serviceaccount
  namespace: "$BOOKSTORE_NAMESPACE"
specs:
- kind: HTTPRouteGroup
  name: bookstore-service-routes
sources:
- kind: ServiceAccount
  name: bookthief-serviceaccount
  namespace: "$BOOKTHIEF_NAMESPACE"
EOF
