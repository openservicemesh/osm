#!/bin/bash

set -aueo pipefail

source .env

SVC=${1:-bookstore}

kubectl delete deployment "$SVC" -n "$K8S_NAMESPACE"  || true

echo -e "Add secrets"
kubectl -n smc delete configmap ca-certpemstore ca-keypemstore || true
kubectl -n smc create configmap ca-certpemstore --from-file=./bin/cert.pem
kubectl -n smc create configmap ca-keypemstore --from-file=./bin/key.pem

echo -e "Deploy $SVC demo service"
cat <<EOF | kubectl apply -f -
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
  template:
    metadata:
      labels:
        app: $SVC
        version: v1
    spec:
      containers:

        - image: "${CTR_REGISTRY}/bookstore:latest"
          imagePullPolicy: Always
          name: $SVC
          ports:
            - containerPort: 80
              name: web
          command: ["/counter"]
          args: ["--path", "./", "--port", "80"]
          env:
            - name: AZMESH_IDENTITY
              value: $SVC

        - image: envoyproxy/envoy-alpine-dev:latest
          imagePullPolicy: Always
          name: envoyproxy
          ports:
            - containerPort: 15000
              name: admin-port
            - containerPort: 15003
              name: mtls-port
          command: ["envoy"]
          args: ["--log-level", "debug", "-c", "/etc/config/${SVC}.yaml"]
          volumeMounts:
           - name: config-volume
             mountPath: /etc/config
           - name: certs-volume
             mountPath: /etc/certs
           # Bootstrap certificates
           - name: ca-certpemstore
             mountPath: /etc/ssl/certs/cert.pem
             subPath: cert.pem
             readOnly: false
           - name: ca-keypemstore
             mountPath: /etc/ssl/certs/key.pem
             subPath: key.pem
             readOnly: false

      volumes:
        - name: config-volume
          configMap:
            name: envoyproxy-config
        - name: certs-volume
          configMap:
            name: certificates-config
        # Bootstrap certificates
        - name: ca-certpemstore
          configMap:
            name: ca-certpemstore
        - name: ca-keypemstore
          configMap:
            name: ca-keypemstore

      imagePullSecrets:
        - name: $CTR_REGISTRY_CREDS_NAME
EOF

kubectl get pods      --no-headers -o wide --selector app=$SVC -n "$K8S_NAMESPACE"
kubectl get endpoints --no-headers -o wide --selector app=$SVC -n "$K8S_NAMESPACE"
kubectl get service                -o wide                     -n "$K8S_NAMESPACE"

for x in $(kubectl get service -n "$K8S_NAMESPACE" --selector app=$SVC --no-headers | awk '{print $1}'); do
    kubectl get service $x -n "$K8S_NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[*].ip}'
done
