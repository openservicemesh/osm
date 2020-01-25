#!/bin/bash

set -aueo pipefail

source .env

kubectl delete pod eds -n "$K8S_NAMESPACE" || true

echo -e "Add secrets"
kubectl -n smc delete configmap ca-certpemstore ca-keypemstore || true
kubectl -n smc create configmap ca-certpemstore --from-file=./bin/cert.pem
kubectl -n smc create configmap ca-keypemstore --from-file=./bin/key.pem

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: eds
  namespace: $K8S_NAMESPACE
  labels:
    app: eds
spec:
  ports:
  - port: 15000
    targetPort: admin-port
    name: eds-envoy-admin-port

  - port: 15124
    targetPort: 15124
    name: eds-port

  selector:
    app: eds

  type: NodePort

---

apiVersion: v1
kind: Pod
metadata:
  name: eds
  namespace: $K8S_NAMESPACE
  labels:
    app: eds

spec:
  containers:
    - image: "${CTR_REGISTRY}/eds:latest"
      imagePullPolicy: Always
      name: curl
      ports:
        - containerPort: 15000
          name: admin-port
        - containerPort: 15124
          name: eds-port

      command: [ "/eds"]
      args:
        - "--kubeconfig"
        - "/kube/config"
        - "--azureAuthFile"
        - "/azure/azureAuth.json"
        - "--subscriptionID"
        - "$AZURE_SUBSCRIPTION"
        - "--verbosity"
        - "7"
        - "--namespace"
        - "smc"
        - "--certpem"
        - "/etc/ssl/certs/cert.pem"
        - "--keypem"
        - "/etc/ssl/certs/key.pem"
        - "--rootcertpem"
        - "/etc/ssl/certs/cert.pem"

      env:
        - name: GRPC_GO_LOG_VERBOSITY_LEVEL
          value: "99"
        - name: GRPC_GO_LOG_SEVERITY_LEVEL
          value: "info"

      volumeMounts:
      - name: kubeconfig
        mountPath: /kube
      - name: azureconfig
        mountPath: /azure
      - name: ca-certpemstore
        mountPath: /etc/ssl/certs/cert.pem
        subPath: cert.pem
        readOnly: false
      - name: ca-keypemstore
        mountPath: /etc/ssl/certs/key.pem
        subPath: key.pem
        readOnly: false

  volumes:
    - name: kubeconfig
      configMap:
        name: kubeconfig
    - name: azureconfig
      configMap:
        name: azureconfig
    - name: ca-certpemstore
      configMap:
        name: ca-certpemstore
    - name: ca-keypemstore
      configMap:
        name: ca-keypemstore

  imagePullSecrets:
    - name: "$CTR_REGISTRY_CREDS_NAME"
EOF
