#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

CURL_NAMESPACE="${CURL_NAMESPACE:-curl}"

echo "Create Hello HTTPRouteGroup"
kubectl apply -f - <<EOF
apiVersion: specs.smi-spec.io/v1alpha4
kind: HTTPRouteGroup
metadata:
  name: hello-service-routes
  namespace: "$CURL_NAMESPACE"
spec:
  matches:
    - name: hello
      methods:
      - GET
      - POST
EOF
