#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

kubectl apply -f https://raw.githubusercontent.com/deislabs/smi-sdk-go/v0.2.0/crds/split.yaml

cat <<EOF > ./demo/policies/TrafficSplit.yaml
apiVersion: split.smi-spec.io/v1alpha2
kind: TrafficSplit
metadata:
  name: bookstore-split
  namespace: "$BOOKSTORE_NAMESPACE"
spec:
  service: "bookstore-mesh.$BOOKSTORE_NAMESPACE"
  backends:

  - service: bookstore-v1
    weight: 100

  - service: bookstore-v2
    weight: 0
EOF

kubectl apply -f ./demo/policies/TrafficSplit.yaml


kubectl apply -f https://raw.githubusercontent.com/deislabs/smi-sdk-go/v0.2.0/crds/access.yaml

cat <<EOF > ./demo/policies/TrafficTarget.yaml
kind: TrafficTarget
apiVersion: access.smi-spec.io/v1alpha1
metadata:
  name: bookbuyer-access-bookstore-v1
  namespace: "$BOOKSTORE_NAMESPACE"

destination:
  kind: ServiceAccount
  name: bookstore-v1-serviceaccount
  namespace: "$BOOKSTORE_NAMESPACE"

specs:
- kind: HTTPRouteGroup
  name: bookstore-service-routes
  matches:
  - buy-a-book
  - books-bought

sources:

- kind: ServiceAccount
  name: bookbuyer-serviceaccount
  namespace: "$BOOKBUYER_NAMESPACE"

#- kind: ServiceAccount
#  name: bookthief-serviceaccount
#  namespace: "$BOOKTHIEF_NAMESPACE"


---

kind: TrafficTarget
apiVersion: access.smi-spec.io/v1alpha1
metadata:
  name: bookbuyer-access-bookstore-v2
  namespace: "$BOOKSTORE_NAMESPACE"
destination:
  kind: ServiceAccount
  name: bookstore-v2-serviceaccount
  namespace: "$BOOKSTORE_NAMESPACE"
specs:
- kind: HTTPRouteGroup
  name: bookstore-service-routes
  matches:
  - buy-a-book
  - books-bought

sources:

- kind: ServiceAccount
  name: bookbuyer-serviceaccount
  namespace: "$BOOKBUYER_NAMESPACE"

#- kind: ServiceAccount
#  name: bookthief-serviceaccount
#  namespace: "$BOOKTHIEF_NAMESPACE"

---

kind: TrafficTarget
apiVersion: access.smi-spec.io/v1alpha1
metadata:
  name: bookstore-access-bookwarehouse
  namespace: "$BOOKWAREHOUSE_NAMESPACE"
destination:
  kind: ServiceAccount
  name: bookwarehouse-serviceaccount
  namespace: "$BOOKWAREHOUSE_NAMESPACE"
specs:
- kind: HTTPRouteGroup
  name: bookwarehouse-service-routes
  matches:
  - restock-books
sources:
- kind: ServiceAccount
  name: bookstore-v1-serviceaccount
  namespace: "$BOOKSTORE_NAMESPACE"
- kind: ServiceAccount
  name: bookstore-v2-serviceaccount
  namespace: "$BOOKSTORE_NAMESPACE"

EOF

kubectl apply -f ./demo/policies/TrafficTarget.yaml

curl -I -X GET http://localhost:8080/reset
curl -I -X GET http://localhost:8081/reset
curl -I -X GET http://localhost:8082/reset
curl -I -X GET http://localhost:8083/reset
