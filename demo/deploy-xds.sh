#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

NAME="ads"
PORT=15128

./demo/deploy-secrets.sh "${NAME}"

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: $NAME
  namespace: $K8S_NAMESPACE
  labels:
    app: $NAME
spec:
  ports:
  - port: $PORT
    targetPort: $PORT
    name: $NAME-port

  selector:
    app: $NAME

  type: NodePort

---

apiVersion: v1
kind: Pod
metadata:
  name: $NAME
  namespace: $K8S_NAMESPACE
  labels:
    app: $NAME

spec:
  containers:
    - image: "${CTR_REGISTRY}/$NAME:latest"
      imagePullPolicy: Always
      name: curl
      ports:
        - containerPort: $PORT
          name: $NAME-port

      command: [ "/$NAME"]
      args:
        - "--kubeconfig"
        - "/kube/config"
        - "--azureAuthFile"
        - "/azure/azureAuth.json"
        - "--subscriptionID"
        - "$AZURE_SUBSCRIPTION"
        - "--verbosity"
        - "17"
        - "--namespace"
        - "smc"
        - "--certpem"
        - "/etc/ssl/certs/cert.pem"
        - "--keypem"
        - "/etc/ssl/certs/key.pem"
        - "--rootcertpem"
        - "/etc/ssl/certs/root-cert.pem"

      volumeMounts:
      - name: kubeconfig
        mountPath: /kube

      - name: azureconfig
        mountPath: /azure

      - name: ca-certpemstore-${NAME}
        mountPath: /etc/ssl/certs/cert.pem
        subPath: cert.pem
        readOnly: false

      - name: ca-keypemstore-${NAME}
        mountPath: /etc/ssl/certs/key.pem
        subPath: key.pem
        readOnly: false

      - name: ca-rootcertpemstore-${NAME}
        mountPath: /etc/ssl/certs/root-cert.pem
        subPath: root-cert.pem
        readOnly: false

      readinessProbe:
        httpGet:
          path: /health/ready
          port: 15000
        initialDelaySeconds: 5
        periodSeconds: 10
      livenessProbe:
        httpGet:
          path: /health/alive
          port: 15000
        initialDelaySeconds: 15
        periodSeconds: 20

  volumes:
    - name: kubeconfig
      configMap:
        name: kubeconfig
    - name: azureconfig
      configMap:
        name: azureconfig
    - name: ca-certpemstore-${NAME}
      configMap:
        name: ca-certpemstore-${NAME}
    - name: ca-rootcertpemstore-${NAME}
      configMap:
        name: ca-rootcertpemstore-${NAME}
    - name: ca-keypemstore-${NAME}
      configMap:
        name: ca-keypemstore-${NAME}

  imagePullSecrets:
    - name: "$CTR_REGISTRY_CREDS_NAME"
EOF
