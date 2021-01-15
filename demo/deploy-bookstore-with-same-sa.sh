#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env
VERSION=${1:-v1}
SVC="bookstore-$VERSION"

kubectl delete deployment "$SVC" -n "$BOOKSTORE_NAMESPACE"  --ignore-not-found

# Create a top level service just for the bookstore domain
echo -e "Deploy bookstore Service"
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: bookstore
  namespace: $BOOKSTORE_NAMESPACE
  labels:
    app: bookstore
spec:
  ports:
  - port: 80
    name: bookstore-port
  selector:
    app: bookstore
EOF

echo -e "Deploy $SVC Service Account"
kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: bookstore
  namespace: $BOOKSTORE_NAMESPACE
EOF

echo -e "Deploy $SVC Service"
kubectl apply -f - <<EOF
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
kubectl apply -f - <<EOF
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
    spec:
      serviceAccountName: bookstore
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
              value: ${SVC}
            - name: BOOKWAREHOUSE_NAMESPACE
              value: ${BOOKWAREHOUSE_NAMESPACE}

          # OSM's mutating webhook will rewrite this liveness probe to /osm-liveness-probe and
          # Envoy will have a dedicated listener on port 15901 for this liveness probe
          livenessProbe:
            httpGet:
              path: /liveness
              port: 80
            initialDelaySeconds: 3
            periodSeconds: 3

          # OSM's mutating webhook will rewrite this readiness probe to /osm-readiness-probe and
          # Envoy will have a dedicated listener on port 15902 for this readiness probe
          readinessProbe:
            failureThreshold: 10
            httpGet:
              path: /readiness
              port: 80
              scheme: HTTP

          # OSM's mutating webhook will rewrite this startup probe to /osm-startup-probe and
          # Envoy will have a dedicated listener on port 15903 for this startup probe
          startupProbe:
            httpGet:
              path: /startup
              port: 80
            failureThreshold: 30
            periodSeconds: 5

      imagePullSecrets:
        - name: $CTR_REGISTRY_CREDS_NAME
EOF

kubectl get pods      --no-headers -o wide --selector app="$SVC" -n "$BOOKSTORE_NAMESPACE"
kubectl get endpoints --no-headers -o wide --selector app="$SVC" -n "$BOOKSTORE_NAMESPACE"
kubectl get service                -o wide                       -n "$BOOKSTORE_NAMESPACE"

for x in $(kubectl get service -n "$BOOKSTORE_NAMESPACE" --selector app="$SVC" --no-headers | awk '{print $1}'); do
    kubectl get service "$x" -n "$BOOKSTORE_NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[*].ip}'
done
