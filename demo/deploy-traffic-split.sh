#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

kubectl apply -f https://raw.githubusercontent.com/deislabs/smi-sdk-go/v0.2.0/crds/split.yaml

kubectl apply -f - <<EOF
apiVersion: split.smi-spec.io/v1alpha2
kind: TrafficSplit
metadata:
  name: bookstore-split
  namespace: "$BOOKSTORE_NAMESPACE"
spec:
  service: "bookstore-mesh.$BOOKSTORE_NAMESPACE"
  backends:

  - service: bookstore-v1
    weight: 1
  - service: bookstore-v2
    weight: 99


EOF
