#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

kubectl apply -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: "$PROMETHEUS_NAMESPACE"
  labels:
    name: "$PROMETHEUS_NAMESPACE"
EOF