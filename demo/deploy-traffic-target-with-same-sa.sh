#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

kubectl apply -f - <<EOF
kind: TrafficTarget
apiVersion: access.smi-spec.io/v1alpha3
metadata:
  name: bookbuyer-access-bookstore
  namespace: "$BOOKSTORE_NAMESPACE"
spec:
  destination:
    kind: ServiceAccount
    name: bookstore
    namespace: "$BOOKSTORE_NAMESPACE"
  rules:
  - kind: HTTPRouteGroup
    name: bookstore-service-routes
    matches:
    - buy-a-book
    - books-bought
  sources:
  - kind: ServiceAccount
    name: bookbuyer
    namespace: "$BOOKBUYER_NAMESPACE"


# TrafficTarget is deny-by-default policy: if traffic from source to destination is not
# explicitly declared in this policy - it will be blocked.
# Should we ever want to allow traffic from bookthief to bookstore the block below needs
# uncommented.

# - kind: ServiceAccount
#   name: bookthief
#   namespace: "$BOOKTHIEF_NAMESPACE"

---

kind: TrafficTarget
apiVersion: access.smi-spec.io/v1alpha3
metadata:
  name: bookstore-access-bookwarehouse
  namespace: "$BOOKWAREHOUSE_NAMESPACE"
spec:
  destination:
    kind: ServiceAccount
    name: bookwarehouse
    namespace: "$BOOKWAREHOUSE_NAMESPACE"
  rules:
  - kind: HTTPRouteGroup
    name: bookwarehouse-service-routes
    matches:
    - restock-books
  sources:
  - kind: ServiceAccount
    name: bookstore
    namespace: "$BOOKSTORE_NAMESPACE"

EOF
