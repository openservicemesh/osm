#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

kubectl delete deployment bookwarehouse -n "$BOOKWAREHOUSE_NAMESPACE"  --ignore-not-found

echo -e "Deploy Bookwarehouse Service Account"
kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: bookwarehouse
  namespace: $BOOKWAREHOUSE_NAMESPACE
EOF

echo -e "Deploy Bookwarehouse Service"
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: bookwarehouse
  namespace: $BOOKWAREHOUSE_NAMESPACE
  labels:
    app: bookwarehouse
spec:
  ports:
  - port: 80
    name: bookwarehouse-port

  selector:
    app: bookwarehouse
EOF

echo -e "Deploy Bookwarehouse Deployment"
kubectl apply -f - <<EOF
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
      serviceAccountName: bookwarehouse

      containers:
        # Main container with APP
        - name: bookwarehouse
          image: "${CTR_REGISTRY}/bookwarehouse:${CTR_TAG}"
          imagePullPolicy: Always
          command: ["/bookwarehouse"]

      imagePullSecrets:
        - name: "$CTR_REGISTRY_CREDS_NAME"
EOF

kubectl get pods      --no-headers -o wide --selector app=bookwarehouse -n "$BOOKWAREHOUSE_NAMESPACE"
kubectl get endpoints --no-headers -o wide --selector app=bookwarehouse -n "$BOOKWAREHOUSE_NAMESPACE"
kubectl get service                -o wide                          -n "$BOOKWAREHOUSE_NAMESPACE"

for x in $(kubectl get service -n "$BOOKWAREHOUSE_NAMESPACE" --selector app=bookwarehouse --no-headers | awk '{print $1}'); do
    kubectl get service "$x" -n "$BOOKWAREHOUSE_NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[*].ip}'
done
