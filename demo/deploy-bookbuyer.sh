#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

WAIT_FOR_OK_SECONDS="${WAIT_FOR_OK_SECONDS:-default 120}"

echo "WAIT_FOR_OK_SECONDS = ${WAIT_FOR_OK_SECONDS}"

NS="${K8S_NAMESPACE}-bookbuyer"

./demo/deploy-secrets.sh "bookbuyer" "$NS"

kubectl delete deployment bookbuyer -n "$NS"  || true

echo -e "Deploy BookBuyer demo service"
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: bookbuyer-serviceaccount
  namespace: $NS
automountServiceAccountToken: false

---

apiVersion: v1
kind: Service
metadata:
  name: bookbuyer
  namespace: "$NS"
  labels:
    app: bookbuyer
spec:
  ports:

  - port: 9999
    name: dummy-unused-port

  selector:
    app: bookbuyer

  type: NodePort

---

apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: bookbuyer
  namespace: "$NS"
spec:
  replicas: 1
  selector:
    matchLabels:
      app: bookbuyer
  template:
    metadata:
      labels:
        app: bookbuyer
        version: v1
    spec:
      serviceAccountName: bookbuyer-serviceaccount
      automountServiceAccountToken: false
      hostAliases:
      - ip: "127.0.0.2"
        hostnames:
        - "${K8S_NAMESPACE}.uswest.mesh"
        - "bookbuyer.mesh"
        - "bookstore.mesh"

      containers:
        # Main container with APP
        - name: bookbuyer
          image: "${CTR_REGISTRY}/bookbuyer:latest"
          imagePullPolicy: Always
          command: ["/bookbuyer"]

          env:
            - name: "WAIT_FOR_OK_SECONDS"
              value: "$WAIT_FOR_OK_SECONDS"


      imagePullSecrets:
        - name: "$CTR_REGISTRY_CREDS_NAME"
EOF

kubectl get pods      --no-headers -o wide --selector app=bookbuyer -n "$NS"
kubectl get endpoints --no-headers -o wide --selector app=bookbuyer -n "$NS"
kubectl get service                -o wide                          -n "$NS"

for x in $(kubectl get service -n "$NS" --selector app=bookbuyer --no-headers | awk '{print $1}'); do
    kubectl get service "$x" -n "$NS" -o jsonpath='{.status.loadBalancer.ingress[*].ip}'
done
