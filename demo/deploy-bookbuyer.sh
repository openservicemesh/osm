#!/bin/bash

set -auexo pipefail

# shellcheck disable=SC1091
source .env

kubectl create namespace "$BOOKBUYER_NAMESPACE" || true
kubectl delete deployment bookbuyer -n "$BOOKBUYER_NAMESPACE"  || true

echo -e "Deploy BookBuyer Service Account"
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: bookbuyer-serviceaccount
  namespace: $BOOKBUYER_NAMESPACE
automountServiceAccountToken: false
EOF

echo -e "Deploy BookBuyer Service"
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: bookbuyer
  namespace: "$BOOKBUYER_NAMESPACE"
  labels:
    app: bookbuyer
spec:
  ports:

  - port: 9999
    name: dummy-unused-port

  selector:
    app: bookbuyer
EOF

echo -e "Deploy BookBuyer Deployment"
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: bookbuyer
  namespace: "$BOOKBUYER_NAMESPACE"
spec:
  replicas: 1
  selector:
    matchLabels:
      app: bookbuyer
  template:
    metadata:
      labels:
        app: bookbuyer
        version: v1
    spec:
      serviceAccountName: bookbuyer-serviceaccount
      automountServiceAccountToken: false

      containers:
        # Main container with APP
        - name: bookbuyer
          image: "${CTR_REGISTRY}/bookbuyer:${CTR_TAG}"
          imagePullPolicy: Always
          command: ["/bookbuyer"]

          env:
            - name: "BOOKSTORE_NAMESPACE"
              value: "$BOOKSTORE_NAMESPACE"


      imagePullSecrets:
        - name: "$CTR_REGISTRY_CREDS_NAME"
EOF

kubectl get pods      --no-headers -o wide --selector app=bookbuyer -n "$BOOKBUYER_NAMESPACE"
kubectl get endpoints --no-headers -o wide --selector app=bookbuyer -n "$BOOKBUYER_NAMESPACE"
kubectl get service                -o wide                          -n "$BOOKBUYER_NAMESPACE"

for x in $(kubectl get service -n "$BOOKBUYER_NAMESPACE" --selector app=bookbuyer --no-headers | awk '{print $1}'); do
    kubectl get service "$x" -n "$BOOKBUYER_NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[*].ip}'
done
