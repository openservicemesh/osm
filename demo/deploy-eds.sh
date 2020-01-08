#!/bin/bash

set -aueo pipefail

source .env

kubectl delete pod eds -n "$K8S_NAMESPACE" || true

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
        - "8"
        - "--namespace"
        - "smc"

      volumeMounts:
      - name: kubeconfig
        mountPath: /kube
      - name: azureconfig
        mountPath: /azure

  volumes:
    - name: kubeconfig
      configMap:
        name: kubeconfig
    - name: azureconfig
      configMap:
        name: azureconfig

  imagePullSecrets:
    - name: "$CTR_REGISTRY_CREDS_NAME"
EOF
