#!/bin/bash

set -aueo pipefail

source .env

kubectl delete service sds -n "$K8S_NAMESPACE" || true
kubectl delete pod sds -n "$K8S_NAMESPACE" || true

echo -e "Add secrets"
kubectl -n smc delete configmap ca-certpemstore ca-keypemstore || true
kubectl -n smc create configmap ca-certpemstore --from-file=./bin/cert.pem
kubectl -n smc create configmap ca-keypemstore --from-file=./bin/key.pem


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

  - port: 15000
    targetPort: admin-port
    name: admin-port

  - port: 15123
    targetPort: 15123
    name: sds-port

  selector:
    app: sds

  type: NodePort

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
        - containerPort: 15000
          name: admin-port
        - containerPort: 15123
          name: sds-port

      command: ["/sds"]
      args:
        - "--kubeconfig"
        - "/kube/config"
        - "--verbosity"
        - "25"
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
    - name: ca-certpemstore
      configMap:
        name: ca-certpemstore
    - name: ca-keypemstore
      configMap:
        name: ca-keypemstore

  imagePullSecrets:
    - name: $CTR_REGISTRY_CREDS_NAME
EOF
