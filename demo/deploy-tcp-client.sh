#!/bin/bash

# This script deploys the resources corresponding to the tcp-client.

set -aueo pipefail

# shellcheck disable=SC1091
source .env

CTR_TAG="${CTR_TAG:-latest}"

echo -e "Create tcp-client service account"
kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tcp-client
  namespace: tcp-demo
EOF

echo -e "Create tcp-client deployment"
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tcp-client-v1
  namespace: tcp-demo
  labels:
    app: tcp-client
    version: v1
spec:
  replicas: 1
  selector:
    matchLabels:
      app: tcp-client
      version: v1
  template:
    metadata:
      labels:
        app: tcp-client
        version: v1
    spec:
      serviceAccountName: tcp-client
      containers:
      - name: tcp-client
        image: "${CTR_REGISTRY}/tcp-client:${CTR_TAG}"
        imagePullPolicy: Always
        command: ["/tcp-client"]
        args: [ "--server-address", "tcp-echo:9000" ]
      imagePullSecrets:
        - name: $CTR_REGISTRY_CREDS_NAME
EOF
