#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

SVC=${1:-bookstore}

./demo/deploy-secrets.sh "$SVC"

kubectl delete deployment "$SVC" -n "$K8S_NAMESPACE"  || true

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
          args: ["--log-level", "debug", "-c", "/etc/config/bootstrap.yaml", "--service-node", "bookstore", "--service-cluster", "bookstore"]
          volumeMounts:
           - name: config-volume
             mountPath: /etc/config
           # Bootstrap certificates
           - name: ca-certpemstore-$SVC
             mountPath: /etc/ssl/certs/cert.pem
             subPath: cert.pem
             readOnly: false
           - name: ca-keypemstore-$SVC
             mountPath: /etc/ssl/certs/key.pem
             subPath: key.pem
             readOnly: false

      volumes:
        - name: config-volume
          configMap:
            name: envoyproxy-config
        # Bootstrap certificates
        - name: ca-certpemstore-$SVC
          configMap:
            name: ca-certpemstore-$SVC
        - name: ca-keypemstore-$SVC
          configMap:
            name: ca-keypemstore-$SVC

      imagePullSecrets:
        - name: $CTR_REGISTRY_CREDS_NAME
EOF

kubectl get pods      --no-headers -o wide --selector app="$SVC" -n "$K8S_NAMESPACE"
kubectl get endpoints --no-headers -o wide --selector app="$SVC" -n "$K8S_NAMESPACE"
kubectl get service                -o wide                     -n "$K8S_NAMESPACE"

for x in $(kubectl get service -n "$K8S_NAMESPACE" --selector app="$SVC" --no-headers | awk '{print $1}'); do
    kubectl get service "$x" -n "$K8S_NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[*].ip}'
done
