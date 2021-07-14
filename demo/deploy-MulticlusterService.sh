#!/bin/bash

NS=bookstore
NAME=bookstore-v1

kubectl delete MultiClusterService -n $NS $NAME

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
    address: 1.1.1.1:8080

  - name: beta
    # certificate
    address: 2.2.2.2:8080

#  ports:
#    - port: 8888
#      protocol: TCP

  serviceAccount: bookstore-v1
EOF
