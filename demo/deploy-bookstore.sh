#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

SVC=${1:-bookstore}

kubectl delete deployment "$SVC" -n "$BOOKSTORE_NAMESPACE"  || true

GIT_HASH=$(git rev-parse --short HEAD)

echo -e "Deploy $SVC demo service"
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: "$SVC-serviceaccount"
  namespace: $BOOKSTORE_NAMESPACE
automountServiceAccountToken: false

---

apiVersion: v1
kind: Service
metadata:
  name: $SVC
  namespace: $BOOKSTORE_NAMESPACE
  labels:
    app: $SVC
spec:
  ports:
  - port: 80
    name: bookstore-port

  selector:
    app: $SVC

  type: NodePort

---

apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: $SVC
  namespace: $BOOKSTORE_NAMESPACE
spec:
  replicas: 1
  selector:
    matchLabels:
      app: $SVC
  template:
    metadata:
      labels:
        app: $SVC
        version: v1
      annotations:
        "osm.io/sidecar-injection": "enabled"
    spec:
      serviceAccountName: "$SVC-serviceaccount"
      automountServiceAccountToken: false
      containers:

        - image: "${CTR_REGISTRY}/bookstore:latest"
          imagePullPolicy: Always
          name: $SVC
          ports:
            - containerPort: 80
              name: web
          command: ["/bookstore"]
          args: ["--path", "./", "--port", "80"]
          env:
            - name: IDENTITY
              value: ${SVC}--${GIT_HASH}

      imagePullSecrets:
        - name: $CTR_REGISTRY_CREDS_NAME
EOF

kubectl get pods      --no-headers -o wide --selector app="$SVC" -n "$BOOKSTORE_NAMESPACE"
kubectl get endpoints --no-headers -o wide --selector app="$SVC" -n "$BOOKSTORE_NAMESPACE"
kubectl get service                -o wide                       -n "$BOOKSTORE_NAMESPACE"

for x in $(kubectl get service -n "$BOOKSTORE_NAMESPACE" --selector app="$SVC" --no-headers | awk '{print $1}'); do
    kubectl get service "$x" -n "$BOOKSTORE_NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[*].ip}'
done
