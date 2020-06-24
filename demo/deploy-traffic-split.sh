#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env


kubectl apply -f - <<EOF
apiVersion: split.smi-spec.io/v1alpha2
kind: TrafficSplit

metadata:
  name: bookstore-split
  namespace: "$BOOKSTORE_NAMESPACE"

spec:
  service: "bookstore.$BOOKSTORE_NAMESPACE"
  backends:

  - service: bookstore-v1
    weight: 0

  - service: bookstore-v2
    weight: 100


EOF
























































































































































































kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: bookstore
  namespace: bookstore
spec:
  ports:
    - port: 9999
EOF
