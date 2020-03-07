#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

SVC=${1:-bookstore}

./demo/deploy-secrets.sh "$SVC"

kubectl delete deployment "$SVC" -n "$K8S_NAMESPACE"  || true

GIT_HASH=$(git rev-parse --short HEAD)

echo -e "Deploy $SVC demo service"
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: "$SVC-serviceaccount"
  namespace: $K8S_NAMESPACE
automountServiceAccountToken: false

---

apiVersion: v1
kind: Service
metadata:
  name: $SVC
  namespace: $K8S_NAMESPACE
  labels:
    app: $SVC
spec:
  ports:
  - port: 89
    targetPort: 15000
    name: admin-port

  - port: 83
    targetPort: 15003
    name: mtls-port

  selector:
    app: $SVC

  type: NodePort

---

apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: $SVC
  namespace: $K8S_NAMESPACE
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

kubectl get pods      --no-headers -o wide --selector app="$SVC" -n "$K8S_NAMESPACE"
kubectl get endpoints --no-headers -o wide --selector app="$SVC" -n "$K8S_NAMESPACE"
kubectl get service                -o wide                       -n "$K8S_NAMESPACE"

for x in $(kubectl get service -n "$K8S_NAMESPACE" --selector app="$SVC" --no-headers | awk '{print $1}'); do
    kubectl get service "$x" -n "$K8S_NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[*].ip}'
done
