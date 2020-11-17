#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

BOOKSTORE_SVC="${BOOKSTORE_SVC:-bookstore}"
BOOKTHIEF_EXPECTED_RESPONSE_CODE="${BOOKTHIEF_EXPECTED_RESPONSE_CODE:-0}"
CI_MAX_ITERATIONS_THRESHOLD="${CI_MAX_ITERATIONS_THRESHOLD:-0}"
CI_CLIENT_CONCURRENT_CONNECTIONS="${CI_CLIENT_CONCURRENT_CONNECTIONS:-1}"
ENABLE_EGRESS="${ENABLE_EGRESS:-false}"
CI_SLEEP_BETWEEN_REQUESTS_SECONDS="${CI_SLEEP_BETWEEN_REQUESTS_SECONDS:-1}"

kubectl delete deployment bookthief -n "$BOOKTHIEF_NAMESPACE"  --ignore-not-found

echo -e "Deploy BookThief demo service"
kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: bookthief
  namespace: $BOOKTHIEF_NAMESPACE

---

apiVersion: v1
kind: Service
metadata:
  name: bookthief
  namespace: "$BOOKTHIEF_NAMESPACE"
  labels:
    app: bookthief
spec:
  ports:

  - port: 9999
    name: dummy-unused-port

  selector:
    app: bookthief

---

apiVersion: apps/v1
kind: Deployment
metadata:
  name: bookthief
  namespace: "$BOOKTHIEF_NAMESPACE"
spec:
  replicas: 1
  selector:
    matchLabels:
      app: bookthief
  template:
    metadata:
      labels:
        app: bookthief
        version: v1
    spec:
      serviceAccountName: bookthief

      containers:
        # Main container with APP
        - name: bookthief
          image: "${CTR_REGISTRY}/bookthief:${CTR_TAG}"
          imagePullPolicy: Always
          command: ["/bookthief"]

          env:
            - name: "BOOKSTORE_NAMESPACE"
              value: "$BOOKSTORE_NAMESPACE"
            - name: "BOOKSTORE_SVC"
              value: "$BOOKSTORE_SVC"
            - name: "BOOKTHIEF_EXPECTED_RESPONSE_CODE"
              value: "$BOOKTHIEF_EXPECTED_RESPONSE_CODE"
            - name: "CI_MAX_ITERATIONS_THRESHOLD"
              value: "$CI_MAX_ITERATIONS_THRESHOLD"
            - name: "ENABLE_EGRESS"
              value: "$ENABLE_EGRESS"

      imagePullSecrets:
        - name: "$CTR_REGISTRY_CREDS_NAME"
EOF

kubectl get pods      --no-headers -o wide --selector app=bookthief -n "$BOOKTHIEF_NAMESPACE"
kubectl get endpoints --no-headers -o wide --selector app=bookthief -n "$BOOKTHIEF_NAMESPACE"
kubectl get service                -o wide                          -n "$BOOKTHIEF_NAMESPACE"

for x in $(kubectl get service -n "$BOOKTHIEF_NAMESPACE" --selector app=bookthief --no-headers | awk '{print $1}'); do
    kubectl get service "$x" -n "$BOOKTHIEF_NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[*].ip}'
done
