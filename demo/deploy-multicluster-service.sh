#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

kubectl apply -f - <<EOF
kind: MultiClusterService
apiVersion: config.openservicemesh.io/v1alpha1
metadata:
  name: bookwarehouse
  namespace: bookwarehouse
spec:
  serviceAccount: bookwarehouse
  globalIP: 192.0.0.12
  clusters:
    - name: cluster-y
      address: "192.0.0.10:15003"
    - name: cluster-z
      address: "192.0.0.11:15003"
  ports: 
    - port: 14001
      protocol: http
EOF
