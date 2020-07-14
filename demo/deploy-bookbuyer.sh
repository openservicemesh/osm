#!/bin/bash

set -auexo pipefail

# shellcheck disable=SC1091
source .env
BOOKSTORE_SVC="${BOOKSTORE_SVC:-bookstore}"
CI_MAX_ITERATIONS_THRESHOLD="${CI_MAX_ITERATIONS_THRESHOLD:-0}"

kubectl delete deployment bookbuyer -n "$BOOKBUYER_NAMESPACE"  || true

echo -e "Deploy BookBuyer Service Account"
kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: bookbuyer
  namespace: $BOOKBUYER_NAMESPACE
EOF

echo -e "Deploy BookBuyer Service"
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: bookbuyer
  namespace: "$BOOKBUYER_NAMESPACE"
  labels:
    app: bookbuyer
spec:
  ports:
    - port: 80
      name: web

  selector:
    app: bookbuyer
EOF

echo -e "Deploy BookBuyer Deployment"
kubectl apply -f - <<EOF
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
      serviceAccountName: bookbuyer

      containers:
        # Main container with APP
        - name: bookbuyer
          image: "${CTR_REGISTRY}/bookbuyer:${CTR_TAG}"
          imagePullPolicy: Always
          command: ["/bookbuyer"]
          ports:
            - containerPort: 80
              name: web

          env:
            - name: "BOOKSTORE_NAMESPACE"
              value: "$BOOKSTORE_NAMESPACE"
            - name: "BOOKSTORE_SVC"
              value: "$BOOKSTORE_SVC"
            - name: "CI_MAX_ITERATIONS_THRESHOLD"
              value: "$CI_MAX_ITERATIONS_THRESHOLD"

      imagePullSecrets:
        - name: "$CTR_REGISTRY_CREDS_NAME"
EOF

kubectl get pods      --no-headers -o wide --selector app=bookbuyer -n "$BOOKBUYER_NAMESPACE"
kubectl get endpoints --no-headers -o wide --selector app=bookbuyer -n "$BOOKBUYER_NAMESPACE"
kubectl get service                -o wide                          -n "$BOOKBUYER_NAMESPACE"

for x in $(kubectl get service -n "$BOOKBUYER_NAMESPACE" --selector app=bookbuyer --no-headers | awk '{print $1}'); do
    kubectl get service "$x" -n "$BOOKBUYER_NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[*].ip}'
done
