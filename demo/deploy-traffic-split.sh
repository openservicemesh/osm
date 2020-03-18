#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

kubectl apply -f https://raw.githubusercontent.com/deislabs/smi-sdk-go/v0.2.0/crds/split.yaml

kubectl apply -f - <<EOF
apiVersion: split.smi-spec.io/v1alpha2
kind: TrafficSplit
metadata:
  name: bookstore.mesh
  namespace: "$BOOKSTORE_NAMESPACE"
spec:
  service: bookstore.mesh
  backends:

  - service: bookstore-1
    weight: 50
  - service: bookstore-2
    weight: 50


EOF
