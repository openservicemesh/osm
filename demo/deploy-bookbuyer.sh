#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env
WAIT_FOR_OK_SECONDS="${WAIT_FOR_OK_SECONDS:-default 120}"

echo "WAIT_FOR_OK_SECONDS = ${WAIT_FOR_OK_SECONDS}"

kubectl create namespace "$BOOKBUYER_NAMESPACE" || true
kubectl delete deployment bookbuyer -n "$BOOKBUYER_NAMESPACE"  || true

echo -e "Deploy BookBuyer demo service"
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: bookbuyer-serviceaccount
  namespace: $BOOKBUYER_NAMESPACE
automountServiceAccountToken: false

---

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

  type: NodePort

---

apiVersion: extensions/v1beta1
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
      # TODO : move prometheus annotations to patch code
      annotations:
        "prometheus.io/scrape": "true"
        "prometheus.io/port": "15010"
        "prometheus.io/path": "/stats/prometheus"
    spec:
      serviceAccountName: bookbuyer-serviceaccount
      automountServiceAccountToken: false
      hostAliases:
      - ip: "127.0.0.2"
        hostnames:
        - "${BOOKBUYER_NAMESPACE}.uswest.mesh"
        - "bookbuyer.mesh"
        - "bookstore.mesh"

      containers:
        # Main container with APP
        - name: bookbuyer
          image: "${CTR_REGISTRY}/bookbuyer:${CTR_TAG}"
          imagePullPolicy: Always
          command: ["/bookbuyer"]

          env:
            - name: "WAIT_FOR_OK_SECONDS"
              value: "$WAIT_FOR_OK_SECONDS"


      imagePullSecrets:
        - name: "$CTR_REGISTRY_CREDS_NAME"
EOF

kubectl get pods      --no-headers -o wide --selector app=bookbuyer -n "$BOOKBUYER_NAMESPACE"
kubectl get endpoints --no-headers -o wide --selector app=bookbuyer -n "$BOOKBUYER_NAMESPACE"
kubectl get service                -o wide                          -n "$BOOKBUYER_NAMESPACE"

for x in $(kubectl get service -n "$BOOKBUYER_NAMESPACE" --selector app=bookbuyer --no-headers | awk '{print $1}'); do
    kubectl get service "$x" -n "$BOOKBUYER_NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[*].ip}'
done
