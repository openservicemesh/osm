#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env
WAIT_FOR_OK_SECONDS="${WAIT_FOR_OK_SECONDS:-default 120}"

echo "WAIT_FOR_OK_SECONDS = ${WAIT_FOR_OK_SECONDS}"

./demo/deploy-secrets.sh "bookthief"

kubectl delete deployment bookthief -n "$BOOKTHIEF_NAMESPACE"  || true

echo -e "Deploy BookThief demo service"
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: bookthief-serviceaccount
  namespace: $BOOKTHIEF_NAMESPACE
automountServiceAccountToken: false

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
      annotations:
        "openservicemesh.io/osm-service": "bookthief"
    spec:
      serviceAccountName: bookthief-serviceaccount
      automountServiceAccountToken: false
      hostAliases:
      - ip: "127.0.0.2"
        hostnames:
        - "${BOOKTHIEF_NAMESPACE}.uswest.mesh"
        - "bookstore.mesh"

      containers:
        # Main container with APP
        - name: bookthief
          image: "${CTR_REGISTRY}/bookthief:${CTR_TAG}"
          imagePullPolicy: Always
          command: ["/bookthief"]

          env:
            - name: "WAIT_FOR_OK_SECONDS"
              value: "$WAIT_FOR_OK_SECONDS"


      imagePullSecrets:
        - name: "$CTR_REGISTRY_CREDS_NAME"
EOF

kubectl get pods      --no-headers -o wide --selector app=bookthief -n "$BOOKTHIEF_NAMESPACE"
kubectl get endpoints --no-headers -o wide --selector app=bookthief -n "$BOOKTHIEF_NAMESPACE"
kubectl get service                -o wide                          -n "$BOOKTHIEF_NAMESPACE"

for x in $(kubectl get service -n "$BOOKTHIEF_NAMESPACE" --selector app=bookthief --no-headers | awk '{print $1}'); do
    kubectl get service "$x" -n "$BOOKTHIEF_NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[*].ip}'
done
