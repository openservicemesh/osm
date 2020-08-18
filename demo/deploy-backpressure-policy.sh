#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

echo "Apply backpressure policy for the bookstore-v1 service"
kubectl apply -f - <<EOF
apiVersion: policy.openservicemesh.io/v1alpha1
kind: Backpressure

metadata:
  name: max-connections-bookstore-v1
  namespace: "${BOOKSTORE_NAMESPACE}"

  labels:
    app: bookstore-v1

spec:
  maxConnections: 5
  maxRequests: 6
  maxPendingRequests: 7
  maxRetries: 8
  maxConnectionPools: 9
EOF

echo "Apply backpressure policy for the bookstore-v2 service"
kubectl apply -f - <<EOF
apiVersion: policy.openservicemesh.io/v1alpha1
kind: Backpressure

metadata:
  name: max-connections-connections-bookstore-v2
  namespace: "${BOOKSTORE_NAMESPACE}"

  labels:
    app: bookstore-v2

spec:
  maxConnections: 10
  maxRequests: 11
  maxPendingRequests: 12
  maxRetries: 13
  maxConnectionPools: 14
EOF
