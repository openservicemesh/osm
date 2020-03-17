#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env
WAIT_FOR_OK_SECONDS="${WAIT_FOR_OK_SECONDS:-default 120}"

echo "WAIT_FOR_OK_SECONDS = ${WAIT_FOR_OK_SECONDS}"

./demo/deploy-secrets.sh "bookthief"

kubectl delete deployment bookthief -n "$K8S_NAMESPACE"  || true

echo -e "Deploy BookThief demo service"
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: bookthief-serviceaccount
  namespace: $K8S_NAMESPACE
automountServiceAccountToken: false

---

apiVersion: v1
kind: Service
metadata:
  name: bookthief
  namespace: "$K8S_NAMESPACE"
  labels:
    app: bookthief
spec:
  ports:

  - port: 9999
    name: dummy-unused-port

  selector:
    app: bookthief

  type: NodePort

---

apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: bookthief
  namespace: "$K8S_NAMESPACE"
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
    spec:
      serviceAccountName: bookthief-serviceaccount
      automountServiceAccountToken: false
      hostAliases:
      - ip: "127.0.0.2"
        hostnames:
        - "${K8S_NAMESPACE}.uswest.mesh"
        - "bookstore.mesh"

      containers:
        # Main container with APP
        - name: bookthief
          image: "${CTR_REGISTRY}/bookthief:latest"
          imagePullPolicy: Always
          command: ["/bookthief"]

          env:
            - name: "WAIT_FOR_OK_SECONDS"
              value: "$WAIT_FOR_OK_SECONDS"


      imagePullSecrets:
        - name: "$CTR_REGISTRY_CREDS_NAME"
EOF

kubectl get pods      --no-headers -o wide --selector app=bookthief -n "$K8S_NAMESPACE"
kubectl get endpoints --no-headers -o wide --selector app=bookthief -n "$K8S_NAMESPACE"
kubectl get service                -o wide                          -n "$K8S_NAMESPACE"

for x in $(kubectl get service -n "$K8S_NAMESPACE" --selector app=bookthief --no-headers | awk '{print $1}'); do
    kubectl get service "$x" -n "$K8S_NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[*].ip}'
done
