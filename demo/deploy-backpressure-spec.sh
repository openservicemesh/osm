#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

echo "Create Backpressure Spec"
kubectl apply -f - <<EOF
apiVersion: policy.openservicemesh.io/v1alpha1
kind: Backpressure

metadata:
  name: max-requests-per-second
  namespace: "${BOOKSTORE_NAMESPACE}"

spec:
  maxRequestsPerConnection: 5
  maxrequestsperconnection: 9

EOF
