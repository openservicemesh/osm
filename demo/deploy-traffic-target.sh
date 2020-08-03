#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

kubectl apply -f - <<EOF
kind: TrafficTarget
apiVersion: access.smi-spec.io/v1alpha2
metadata:
  name: bookbuyer-access-bookstore-v1
  namespace: "$BOOKSTORE_NAMESPACE"
spec:
  destination:
    kind: ServiceAccount
    name: bookstore-v1
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

---

kind: TrafficTarget
apiVersion: access.smi-spec.io/v1alpha2
metadata:
  name: bookbuyer-access-bookstore-v2
  namespace: "$BOOKSTORE_NAMESPACE"
spec:
  destination:
    kind: ServiceAccount
    name: bookstore-v2
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

---

kind: TrafficTarget
apiVersion: access.smi-spec.io/v1alpha2
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
    name: bookstore-v1
    namespace: "$BOOKSTORE_NAMESPACE"
  - kind: ServiceAccount
    name: bookstore-v2
    namespace: "$BOOKSTORE_NAMESPACE"

EOF
