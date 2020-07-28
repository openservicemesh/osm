#!/bin/bash

helm repo add dapr https://daprio.azurecr.io/helm/v1/repo
helm repo update

kubectl create namespace dapr-system

helm install dapr dapr/dapr --namespace dapr-system

kubectl create namespace redis-system

helm repo add bitnami https://charts.bitnami.com/bitnami
helm install redis bitnami/redis --namespace calculator


kubectl delete namespace calculator || true
kubectl create namespace calculator
kubectl apply -f - <<EOF
apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: statestore
  namespace: calculator
spec:
  type: state.redis

  metadata:

  - name: redisHost
    value: redis-master:6379

  - name: redisPassword
    value: $(kubectl get secret --namespace default redis -o jsonpath="{.data.redis-password}" | base64 --decode)
EOF

kubectl apply -f .
