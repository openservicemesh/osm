#!/bin/bash

set -auexo pipefail

# shellcheck disable=SC1091
source .env

kubectl delete deployment bookwarehouse -n "$BOOKWAREHOUSE_NAMESPACE"  || true

echo -e "Deploy Bookwarehouse Service Account"
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: bookwarehouse-serviceaccount
  namespace: $BOOKWAREHOUSE_NAMESPACE
automountServiceAccountToken: false
EOF

echo -e "Deploy Bookwarehouse Service"
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: bookwarehouse
  namespace: $BOOKWAREHOUSE_NAMESPACE
  labels:
    app: bookwarehouse
spec:
  ports:
  - port: 8080
    name: bookwarehouse-port

  selector:
    app: bookwarehouse
EOF

echo -e "Deploy Bookwarehouse Deployment"
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: bookwarehouse
  namespace: "$BOOKWAREHOUSE_NAMESPACE"
spec:
  replicas: 1
  selector:
    matchLabels:
      app: bookwarehouse
  template:
    metadata:
      labels:
        app: bookwarehouse
        version: v1
    spec:
      serviceAccountName: bookwarehouse-serviceaccount
      automountServiceAccountToken: false

      containers:
        # Main container with APP
        - name: bookwarehouse
          image: "${CTR_REGISTRY}/bookwarehouse:${CTR_TAG}"
          imagePullPolicy: Always
          command: ["/bookwarehouse"]

          env:
            - name: "OSM_HUMAN_DEBUG_LOG"
              value: "true"

      imagePullSecrets:
        - name: "$CTR_REGISTRY_CREDS_NAME"
EOF

kubectl get pods      --no-headers -o wide --selector app=bookwarehouse -n "$BOOKWAREHOUSE_NAMESPACE"
kubectl get endpoints --no-headers -o wide --selector app=bookwarehouse -n "$BOOKWAREHOUSE_NAMESPACE"
kubectl get service                -o wide                          -n "$BOOKWAREHOUSE_NAMESPACE"

for x in $(kubectl get service -n "$BOOKWAREHOUSE_NAMESPACE" --selector app=bookwarehouse --no-headers | awk '{print $1}'); do
    kubectl get service "$x" -n "$BOOKWAREHOUSE_NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[*].ip}'
done
