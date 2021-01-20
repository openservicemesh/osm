#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

echo "Create Bookstore HTTPRouteGroup"
kubectl apply -f - <<EOF
apiVersion: specs.smi-spec.io/v1alpha4
kind: HTTPRouteGroup
metadata:
  name: bookstore-service-routes
  namespace: "$BOOKSTORE_NAMESPACE"
spec:
  matches:
  - name: books-bought
    pathRegex: /books-bought
    methods:
    - GET
    headers:
    - "user-agent": ".*-http-client/*.*"
    - "client-app": "bookbuyer"
  - name: buy-a-book
    pathRegex: ".*a-book.*new"
    methods:
    - GET
  - name: update-books-bought
    pathRegex: /update-books-bought
    methods:
    - POST
EOF


echo "Create Bookwarehouse HTTPRouteGroup"
kubectl apply -f - <<EOF
apiVersion: specs.smi-spec.io/v1alpha4
kind: HTTPRouteGroup
metadata:
  name: bookwarehouse-service-routes
  namespace: "$BOOKWAREHOUSE_NAMESPACE"
spec:
  matches:
    - name: restock-books
      methods:
      - POST
      headers:
      - host: "bookwarehouse.$BOOKWAREHOUSE_NAMESPACE"
EOF
