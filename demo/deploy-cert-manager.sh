#!/bin/bash

# shellcheck disable=SC1091
source .env

CERT_MANAGER_VERSION="${CERT_MANAGER_VERSION:-0.16.1}"

apply_cert-manager_bootstrap_manifests() {
  kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1alpha2
kind: Issuer
metadata:
  name: selfsigned
  namespace: $K8S_NAMESPACE
spec:
  selfSigned: {}
---
apiVersion: cert-manager.io/v1alpha2
kind: Certificate
metadata:
  name: osm-ca
  namespace: $K8S_NAMESPACE
spec:
  isCA: true
  duration: 2160h # 90d
  secretName: osm-ca-bundle
  commonName: osm-system
  issuerRef:
    name: selfsigned
    kind: Issuer
    group: cert-manager.io
---
apiVersion: cert-manager.io/v1alpha2
kind: Issuer
metadata:
  name: osm-ca
  namespace: $K8S_NAMESPACE
spec:
  ca:
    secretName: osm-ca-bundle
EOF

return $?
}

# shellcheck disable=SC2086
kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v$CERT_MANAGER_VERSION/cert-manager.yaml

kubectl rollout status deploy -n cert-manager cert-manager
kubectl rollout status deploy -n cert-manager cert-manager-cainjector
kubectl rollout status deploy -n cert-manager cert-manager-webhook


max=15

for x in $(seq 1 $max); do
    apply_cert-manager_bootstrap_manifests
    res=$?

    if [ $res -eq 0 ]; then
        exit 0
    fi

    echo "[${x}] cert-manager not ready" && sleep 5
done

echo "Failed to deploy cert-manager and bootstrap manifests in time"
exit 1
