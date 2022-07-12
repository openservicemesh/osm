#!/bin/bash

# This script deploys the resources corresponding to the tcp-echo service.

set -aueo pipefail

# shellcheck disable=SC1091
source .env

CTR_TAG="${CTR_TAG:-latest}"

echo -e "Create tcp-echo service"
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: tcp-echo
  namespace: tcp-demo
  labels:
    app: tcp-echo
spec:
  ports:
  - name: tcp
    port: 9000
    appProtocol: tcp
  selector:
    app: tcp-echo
EOF

echo -e "Create tcp-echo service account"
kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tcp-echo
  namespace: tcp-demo
EOF

echo -e "Create tcp-echo deployment"
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tcp-echo-v1
  namespace: tcp-demo
  labels:
    app: tcp-echo
    version: v1
spec:
  replicas: 1
  selector:
    matchLabels:
      app: tcp-echo
      version: v1
  template:
    metadata:
      labels:
        app: tcp-echo
        version: v1
    spec:
      serviceAccountName: tcp-echo
      containers:
      - name: tcp-echo-server
        image: "${CTR_REGISTRY}/tcp-echo-server:${CTR_TAG}"
        imagePullPolicy: Always
        command: ["/tcp-echo-server"]
        args: [ "--port", "9000" ]
        ports:
        - containerPort: 9000
          name: tcp-echo-server
      imagePullSecrets:
        - name: $CTR_REGISTRY_CREDS_NAME
EOF
