#!/bin/bash

set -aueo pipefail

source .env

kubectl delete deployment bookbuyer -n "$K8S_NAMESPACE"  || true

echo -e "Add secrets"
kubectl -n smc delete configmap ca-certpemstore ca-keypemstore || true
kubectl -n smc create configmap ca-certpemstore --from-file=./bin/cert.pem
kubectl -n smc create configmap ca-keypemstore --from-file=./bin/key.pem

echo -e "Deploy BookBuyer demo service"
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: bookbuyer
  namespace: "$K8S_NAMESPACE"
  labels:
    app: bookbuyer
spec:
  ports:

  - port: 15000
    targetPort: admin-port
    name: bookbuyer-envoy-admin-port

  selector:
    app: bookbuyer

  type: NodePort

---

apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: bookbuyer
  namespace: "$K8S_NAMESPACE"
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: bookbuyer
        version: v1
    spec:
      hostAliases:
      - ip: "127.0.0.1"
        hostnames:
        - "${K8S_NAMESPACE}.uswest.mesh"
        - "bookbuyer.mesh"
        - "bookstore.mesh"

      containers:
        # Main container with APP
        - name: bookbuyer
          image: "${CTR_REGISTRY}/bookbuyer:latest"
          imagePullPolicy: Always
          volumeMounts:
           - name: config-volume
             mountPath: /etc/config
           - name: certs-volume
             mountPath: /etc/certs


        # Sidecar with Envoy PROXY
        - name: envoyproxy
          image: envoyproxy/envoy-alpine-dev:latest
          imagePullPolicy: Always
          securityContext:
            runAsUser: 1337
          ports:
            - containerPort: 15000
              name: admin-port
          command: ["envoy"]
          args: ["--log-level", "debug", "-c", "/etc/config/bookbuyer.yaml"]
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

      initContainers:
        - name: proxyinit
          image: "${CTR_REGISTRY}/init"
          securityContext:
            capabilities:
              add:
                - NET_ADMIN
          env:
            - name: "AZMESH_IDENTITY"
              value: "bookbuyer"
            - name: "AZMESH_START_ENABLED"
              value: "1"
            - name: "AZMESH_IGNORE_UID"
              value: "1337"
            - name: "AZMESH_ENVOY_INGRESS_PORT"
              value: "15000"
            - name: "AZMESH_ENVOY_EGRESS_PORT"
              value: "15001"
            - name: "AZMESH_APP_PORTS"
              value: "8080"
            - name: "AZMESH_EGRESS_IGNORED_IP"
              value: "169.254.169.254"

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
        - name: "$CTR_REGISTRY_CREDS_NAME"
EOF

kubectl get pods      --no-headers -o wide --selector app=bookbuyer -n "$K8S_NAMESPACE"
kubectl get endpoints --no-headers -o wide --selector app=bookbuyer -n "$K8S_NAMESPACE"
kubectl get service                -o wide                          -n "$K8S_NAMESPACE"

for x in $(kubectl get service -n "$K8S_NAMESPACE" --selector app=bookbuyer --no-headers | awk '{print $1}'); do
    kubectl get service $x -n "$K8S_NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[*].ip}'
done
