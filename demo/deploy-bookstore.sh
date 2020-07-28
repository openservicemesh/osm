#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env
VERSION=${1:-v1}
SVC="bookstore-$VERSION"
NS="${BOOKSTORE_NAMESPACE}-${VERSION}"

kubectl delete deployment "$SVC" -n "$NS" --ignore-not-found

# Create a top level service just for the bookstore domain
echo -e "Deploy bookstore Service"
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: bookstore
  namespace: $NS
  labels:
    app: bookstore
spec:
  ports:
  - port: 8080
    name: bookstore-port
  selector:
    app: bookstore
EOF

echo -e "Deploy $SVC Service Account"
kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: "$SVC"
  namespace: $NS
EOF

echo -e "Deploy $SVC Service"
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: $SVC
  namespace: $NS
  labels:
    app: $SVC
spec:
  ports:
  - port: 8080
    name: bookstore-port

  selector:
    app: $SVC
EOF

echo -e "Deploy $SVC Deployment"
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: $SVC
  namespace: $NS
spec:
  replicas: 1
  selector:
    matchLabels:
      app: $SVC
      version: $VERSION
  template:
    metadata:
      labels:
        app: $SVC
        version: $VERSION
    spec:
      serviceAccountName: "$SVC"
      containers:
        - image: "${CTR_REGISTRY}/bookstore:${CTR_TAG}"
          imagePullPolicy: Always
          name: $SVC
          ports:
            - containerPort: 8080
              name: web
          command: ["/bookstore"]
          args: ["--path", "./", "--port", "8080"]
          env:
            - name: IDENTITY
              value: ${SVC}
            - name: BOOKWAREHOUSE_NAMESPACE
              value: ${BOOKWAREHOUSE_NAMESPACE}

      imagePullSecrets:
        - name: $CTR_REGISTRY_CREDS_NAME
EOF

kubectl get pods      --no-headers -o wide --selector app="$SVC" -n "$NS"
kubectl get endpoints --no-headers -o wide --selector app="$SVC" -n "$NS"
kubectl get service                -o wide                       -n "$NS"

for x in $(kubectl get service -n "$NS" --selector app="$SVC" --no-headers | awk '{print $1}'); do
    kubectl get service "$x" -n "$NS" -o jsonpath='{.status.loadBalancer.ingress[*].ip}'
done
