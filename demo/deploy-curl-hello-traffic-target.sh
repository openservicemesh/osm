#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

CURL_NAMESPACE="${CURL_NAMESPACE:-curl}"
HELLO_NAMESPACE="${HELLO_NAMESPACE:-hello}"

kubectl apply -f - <<EOF
kind: TrafficTarget
apiVersion: access.smi-spec.io/v1alpha3
metadata:
  name: curl-access-hello
  namespace: "$HELLO_NAMESPACE"
spec:
  destination:
    kind: ServiceAccount
    name: hello
    namespace: "$HELLO_NAMESPACE"
  rules:
  - kind: HTTPRouteGroup
    name: hello-service-routes
    matches:
    - hello
  sources:
  - kind: ServiceAccount
    name: curl
    namespace: "$CURL_NAMESPACE"
EOF
