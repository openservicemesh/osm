#!/bin/bash

set -auexo pipefail

NS=bookstore
NAME=bookstore

kubectl delete MultiClusterService -n $NS $NAME || true

kubectl apply -f - <<EOF
apiVersion: config.openservicemesh.io/v1alpha1
kind: MultiClusterService

metadata:
  namespace: $NS
  name: $NAME

spec:
  clusters:

  - name: alpha

    # TODO: This address and port number must be updated
    address: 1.1.1.1:1111

  - name: beta

    # TODO: This address and port number must be updated
    address: 2.2.2.2:2222

  serviceAccount: bookstore-v1
EOF
