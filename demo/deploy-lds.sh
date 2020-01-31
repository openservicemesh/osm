#!/bin/bash

set -aueo pipefail

source .env

kubectl delete pod lds -n "$K8S_NAMESPACE" || true

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: lds
  namespace: $K8S_NAMESPACE
  labels:
    app: lds
spec:
  ports:
  - port: 15000
    targetPort: admin-port
    name: lds-envoy-admin-port

  - port: 15125
    targetPort: 15125
    name: lds-port

  selector:
    app: lds

  type: NodePort

---

apiVersion: v1
kind: Pod
metadata:
  name: lds
  namespace: $K8S_NAMESPACE
  labels:
    app: lds

spec:
  containers:
    - image: "${CTR_REGISTRY}/lds:latest"
      imagePullPolicy: Always
      name: curl
      ports:
        - containerPort: 15000
          name: admin-port
        - containerPort: 15125
          name: lds-port

      command: [ "/lds"]
      args:
        - "--kubeconfig"
        - "/kube/config"
        - "--verbosity"
        - "25"
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
