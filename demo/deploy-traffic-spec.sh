#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

kubectl apply -f https://raw.githubusercontent.com/deislabs/smi-sdk-go/v0.3.0/crds/specs.yaml

echo "Create HTTPRouteGroup"
kubectl apply -f - <<EOF
apiVersion: specs.smi-spec.io/v1alpha2
kind: HTTPRouteGroup
metadata:
  name: bookstore-service-routes
  namespace: "$BOOKSTORE_NAMESPACE"
matches:
- name: books-bought
  pathRegex: /books-bought
  methods:
  - GET
- name: buy-a-book
  pathRegex: ".*a-book.*new"
  methods: 
  - GET
- name: update-books-bought
  pathRegex: /update-books-bought
  methods:
  - POST
EOF
