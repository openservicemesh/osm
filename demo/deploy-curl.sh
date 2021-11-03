#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

CURL_NAMESPACE="${CURL_NAMESPACE:-curl}"

kubectl delete deployment curl -n "$CURL_NAMESPACE"  --ignore-not-found

echo -e "Deploy Curl Service Account"
kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: curl
  namespace: $CURL_NAMESPACE
EOF

echo -e "Deploy Curl Deployment"
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: curl
  namespace: "$CURL_NAMESPACE"
spec:
  replicas: 1
  selector:
    matchLabels:
      app: curl
      version: v1
  template:
    metadata:
      labels:
        app: curl
        version: v1
    spec:
      serviceAccountName: curl
      nodeSelector:
        kubernetes.io/arch: amd64
        kubernetes.io/os: linux
      containers:
        - name: curl
          image: curlimages/curl
          imagePullPolicy: Always
          command: ["sleep", "365d"]
EOF

kubectl -n "$CURL_NAMESPACE" rollout status deployment curl

kubectl get pods      --no-headers -o wide --selector app=curl -n "$CURL_NAMESPACE"
kubectl get endpoints --no-headers -o wide --selector app=curl -n "$CURL_NAMESPACE"
kubectl get service                -o wide                          -n "$CURL_NAMESPACE"

for x in $(kubectl get service -n "$CURL_NAMESPACE" --selector app=curl --no-headers | awk '{print $1}'); do
    kubectl get service "$x" -n "$CURL_NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[*].ip}'
done
