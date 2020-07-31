#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

kubectl apply -f - <<EOF
apiVersion: specs.smi-spec.io/v1alpha3
kind: TrafficSplit
metadata:
  name: bookstore-split
  namespace: "$BOOKSTORE_NAMESPACE"
spec:
  service: "bookstore.$BOOKSTORE_NAMESPACE"
  backends:
  - service: bookstore-v1
    weight: 50
  - service: bookstore-v2
    weight: 50


EOF
