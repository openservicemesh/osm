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
    # certificate
    address: 1.99.99.1:7777

  - name: beta
    # certificate
    address: 2.2.2.2:5555

#  ports:
#    - port: 8888
#      protocol: TCP

  serviceAccount: bookstore-v1
EOF
