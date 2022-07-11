#!/bin/bash

# Source: https://raw.githubusercontent.com/hashicorp/microsoft-oss-conference/ffdea87a63a115ca6a8ecaf0a02f1b605ac853bf/kubernetes/vault.yaml

# shellcheck disable=SC1091
source .env

kubectl delete deployment vault -n "$K8S_NAMESPACE" --ignore-not-found
kubectl delete pod vault -n "$K8S_NAMESPACE" --ignore-not-found
kubectl delete service vault -n "$K8S_NAMESPACE" --ignore-not-found

kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: vault
  namespace: $K8S_NAMESPACE
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
        command: ["/bin/sh","-c"]
        args:
          - |
            # The TTL for the expiration of CA certificate must be beyond that of the longest
            # TTL for a certificate issued by OSM. The longest TTL for a certificate issued
            # within OSM is 87600h.

            # Start the Vault Server
            vault server -dev -dev-listen-address=0.0.0.0:8200 -dev-root-token-id=$VAULT_TOKEN & sleep 1;

            # Make the token available to the following commands
            echo $VAULT_TOKEN>~/.vault-token;

            # Enable PKI secrets engine
            vault secrets enable pki;

            # Set the max allowed lease for a certificate to a decade
            vault secrets tune -max-lease-ttl=87700h pki;

            # Set the URLs (See: https://www.vaultproject.io/docs/secrets/pki#set-url-configuration)
            vault write pki/config/urls issuing_certificates='http://127.0.0.1:8200/v1/pki/ca' crl_distribution_points='http://127.0.0.1:8200/v1/pki/crl';

            # Configure a role for OSM (See: https://www.vaultproject.io/docs/secrets/pki#configure-a-role)
            vault write pki/roles/${VAULT_ROLE} allow_any_name=true allow_subdomains=true max_ttl=87700h;

            # Create the root certificate (See: https://www.vaultproject.io/docs/secrets/pki#setup)
            vault write pki/root/generate/internal common_name='osm.root' ttl='87700h';
            tail /dev/random;
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
        - name: VAULT_ADDR
          value: "http://localhost:8200"
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
  namespace: $K8S_NAMESPACE
  labels:
    app: vault
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
