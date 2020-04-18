#!/bin/bash

# Source: https://raw.githubusercontent.com/hashicorp/microsoft-oss-conference/ffdea87a63a115ca6a8ecaf0a02f1b605ac853bf/kubernetes/vault.yaml

kubectl delete deployment vault
kubectl delete pod vault
kubectl delete service vault

cat<<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: vault
  labels:
    app: vault
spec:
  replicas: 1
  selector:
    matchLabels:
      app: vault
  template:
    metadata:
      labels:
        app: vault
    spec:
      terminationGracePeriodSeconds: 10
      containers:
      - name: vault
        image: registry.hub.docker.com/library/vault:1.4.0
        imagePullPolicy: Always
        # args: ['server', '-dev']
        command: ["/bin/sh","-c"]
        args: ["vault server -dev -dev-listen-address=0.0.0.0:8200 -dev-root-token-id='xxx' & sleep 1; echo 'xxx'>~/.vault-token; VAULT_ADDR=http://localhost:8200 vault secrets enable pki; VAULT_ADDR=http://localhost:8200 vault secrets tune -max-lease-ttl=87600h pki; VAULT_ADDR=http://localhost:8200 vault write pki/config/urls issuing_certificates='http://127.0.0.1:8200/v1/pki/ca' crl_distribution_points='http://127.0.0.1:8200/v1/pki/crl'; VAULT_ADDR=http://localhost:8200 vault write pki/roles/open-service-mesh allow_any_name=true allow_subdomains=true; tail /dev/random"]
        securityContext:
          capabilities:
            add: ['IPC_LOCK']
        ports:
        - containerPort: 8200
          name: vault-port
          protocol: TCP
        - containerPort: 8201
          name: cluster-port
          protocol: TCP
        env:
        - name: POD_IP_ADDR
          valueFrom:
            fieldRef:
              fieldPath: status.podIP
        - name: VAULT_LOCAL_CONFIG
          value: |
            api_addr     = "http://127.0.0.1:8200"
            cluster_addr = "http://${POD_IP_ADDR}:8201"
        - name: VAULT_DEV_ROOT_TOKEN_ID
          value: "root" ## THIS IS NOT A PRODUCTION DEPLOYMENT OF VAULT!
        readinessProbe:
          httpGet:
            path: /v1/sys/health
            port: 8200
            scheme: HTTP
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: vault
  labels:
    app: vault
  annotations:
    service.beta.kubernetes.io/azure-load-balancer-internal: "true"
spec:
  type: LoadBalancer
  selector:
    app: vault
  ports:
  - name: vault-port
    port: 8200
    targetPort: 8200
    protocol: TCP
EOF

kubectl logs -f $(kubectl get pods --selector app=vault --no-headers=true | awk '{print $1}')


exit 0

kubectl delete pod vault


cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: vault
  labels:
    app: vault
spec:
  containers:
  - name: vault
    image: vault
    command: ["/bin/sh","-c"]
    args: ["vault server -dev -dev-root-token-id='xxx' & sleep 1; echo 'xxx'>~/.vault-token; VAULT_ADDR=http://localhost:8200 vault secrets enable pki; VAULT_ADDR=http://localhost:8200 vault secrets tune -max-lease-ttl=87600h pki; VAULT_ADDR=http://localhost:8200 vault write pki/config/urls issuing_certificates='http://127.0.0.1:8200/v1/pki/ca' crl_distribution_points='http://127.0.0.1:8200/v1/pki/crl'; tail /dev/random"]
    ports:
    - containerPort: 8200
EOF


cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: vault
  labels:
    app: vault
spec:
  ports:
  - port: 18200
    targetPort: 8200
  selector:
    app: vault
EOF
