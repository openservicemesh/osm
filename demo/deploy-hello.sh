#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

HELLO_NAMESPACE="${HELLO_NAMESPACE:-hello}"

kubectl delete deployment hello -n "$HELLO_NAMESPACE"  --ignore-not-found

echo -e "Deploy Hello Service Account"
kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: hello
  namespace: $HELLO_NAMESPACE
EOF

echo -e "Deploy Hello Service"
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: hello
  namespace: $HELLO_NAMESPACE
  labels:
    app: hello
spec:
  ports:
  - port: 80
    targetPort: 80
    protocol: TCP

  selector:
    app: hello
EOF

echo -e "Deploy Hello Deployment"
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hello
  namespace: "$HELLO_NAMESPACE"
spec:
  replicas: 1
  selector:
    matchLabels:
      app: hello
      version: v1
  template:
    metadata:
      labels:
        app: hello
        version: v1
    spec:
      serviceAccountName: hello
      nodeSelector:
        kubernetes.io/arch: amd64
        kubernetes.io/os: linux
      containers:
        - name: hello
          image: tutum/hello-world
          imagePullPolicy: Always
          ports:
          - containerPort: 80
EOF

kubectl -n "$HELLO_NAMESPACE" rollout status deployment hello

kubectl get pods      --no-headers -o wide --selector app=hello -n "$HELLO_NAMESPACE"
kubectl get endpoints --no-headers -o wide --selector app=hello -n "$HELLO_NAMESPACE"
kubectl get service                -o wide                          -n "$HELLO_NAMESPACE"

for x in $(kubectl get service -n "$HELLO_NAMESPACE" --selector app=hello --no-headers | awk '{print $1}'); do
    kubectl get service "$x" -n "$HELLO_NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[*].ip}'
done
