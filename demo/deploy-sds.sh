#!/bin/bash

set -aueo pipefail

source .env

kubectl delete service sds -n "$K8S_NAMESPACE" || true
kubectl delete pod sds -n "$K8S_NAMESPACE" || true

echo -e "Deploy sds service"
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: sds
  namespace: $K8S_NAMESPACE
  labels:
    app: sds
spec:
  ports:

  - port: 15123
    targetPort: 15123
    name: sds-port

  selector:
    app: sds

---

apiVersion: v1
kind: Pod
metadata:
  name: sds
  namespace: $K8S_NAMESPACE
  labels:
    app: sds
spec:
  containers:
    - image: "${CTR_REGISTRY}/sds:latest"
      imagePullPolicy: Always
      name: sds
      ports:
        - containerPort: 15123
      command: ["/sds"]
      args: ["--verbosity", "9"]

  imagePullSecrets:
    - name: $CTR_REGISTRY_CREDS_NAME
EOF
