#!/bin/bash

set -auexo pipefail

# shellcheck disable=SC1091
source .env
VERSION=${1:-v1}
SVC="bookstore-$VERSION"

kubectl delete deployment "$SVC" -n "$BOOKSTORE_NAMESPACE"  || true

GIT_HASH=$(git rev-parse --short HEAD)

# Create a top level service just for the bookstore.mesh domain
echo -e "Deploy bookstore-mesh Service"
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: bookstore-mesh
  namespace: $BOOKSTORE_NAMESPACE
  labels:
    app: bookstore-mesh
spec:
  ports:
  - port: 80
    name: bookstore-port
  selector:
    app: bookstore-mesh
EOF

echo -e "Deploy $SVC Service Account"
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: "$SVC"
  namespace: $BOOKSTORE_NAMESPACE
EOF

echo -e "Deploy $SVC Service"
cat <<EOF | kubectl apply -f -
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
EOF

echo -e "Deploy $SVC Deployment"
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: $SVC
  namespace: $BOOKSTORE_NAMESPACE
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
      annotations:
        "openservicemesh.io/sidecar-injection": "enabled"
    spec:
      serviceAccountName: "$SVC"
      containers:
        - image: "${CTR_REGISTRY}/bookstore:${CTR_TAG}"
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
            - name: BOOKWAREHOUSE_NAMESPACE
              value: ${BOOKWAREHOUSE_NAMESPACE}
            - name: "OSM_HUMAN_DEBUG_LOG"
              value: "true"

      imagePullSecrets:
        - name: $CTR_REGISTRY_CREDS_NAME
EOF

kubectl get pods      --no-headers -o wide --selector app="$SVC" -n "$BOOKSTORE_NAMESPACE"
kubectl get endpoints --no-headers -o wide --selector app="$SVC" -n "$BOOKSTORE_NAMESPACE"
kubectl get service                -o wide                       -n "$BOOKSTORE_NAMESPACE"

for x in $(kubectl get service -n "$BOOKSTORE_NAMESPACE" --selector app="$SVC" --no-headers | awk '{print $1}'); do
    kubectl get service "$x" -n "$BOOKSTORE_NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[*].ip}'
done
