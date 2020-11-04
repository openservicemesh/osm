#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env
BOOKSTORE_SVC="${BOOKSTORE_SVC:-bookstore}"
CI_MAX_ITERATIONS_THRESHOLD="${CI_MAX_ITERATIONS_THRESHOLD:-0}"
CI_CLIENT_CONCURRENT_CONNECTIONS="${CI_CLIENT_CONCURRENT_CONNECTIONS:-1}"
ENABLE_EGRESS="${ENABLE_EGRESS:-false}"
EGRESS_EXPECTED_RESPONSE_CODE="${EGRESS_EXPECTED_RESPONSE_CODE:-404}"
CI_SLEEP_BETWEEN_REQUESTS_SECONDS="${CI_SLEEP_BETWEEN_REQUESTS_SECONDS:-1}"

kubectl delete deployment flaneur -n "$BOOKBUYER_NAMESPACE"  --ignore-not-found


echo -e "Deploy BookBuyer Service Account"
kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: flaneur
  namespace: $BOOKBUYER_NAMESPACE
EOF


echo -e "Deploy Flaneur Deployment"
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: flaneur
  namespace: $BOOKBUYER_NAMESPACE
spec:
  replicas: 1
  selector:
    matchLabels:
      app: flaneur
  template:
    metadata:
      labels:
        app: flaneur
        version: v1
    spec:
      serviceAccountName: flaneur

      containers:
        # Main container with APP
        - name: flaneur
          image: "${CTR_REGISTRY}/bookbuyer:${CTR_TAG}"
          imagePullPolicy: Always
          command: ["/bookbuyer"]

          env:
            - name: "BOOKSTORE_NAMESPACE"
              value: "$BOOKSTORE_NAMESPACE"
            - name: "BOOKSTORE_SVC"
              value: "$BOOKSTORE_SVC"
            - name: "CI_MAX_ITERATIONS_THRESHOLD"
              value: "$CI_MAX_ITERATIONS_THRESHOLD"
            - name: "ENABLE_EGRESS"
              value: "$ENABLE_EGRESS"
            - name: "EGRESS_EXPECTED_RESPONSE_CODE"
              value: "$EGRESS_EXPECTED_RESPONSE_CODE"
            - name: "CI_CLIENT_CONCURRENT_CONNECTIONS"
              value: "$CI_CLIENT_CONCURRENT_CONNECTIONS"
            - name: "CI_SLEEP_BETWEEN_REQUESTS_SECONDS"
              value: "$CI_SLEEP_BETWEEN_REQUESTS_SECONDS"

      imagePullSecrets:
        - name: "$CTR_REGISTRY_CREDS_NAME"
EOF
