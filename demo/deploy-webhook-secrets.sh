#!/usr/bin/env bash
# shellcheck disable=SC1091
set -euo pipefail

source .env

basedir="$(dirname "$0")/"
keydir="$basedir/webhook-certs"

# Generate keys into a temporary directory.
echo "Generating TLS keys ..."
"${basedir}/gen-keys.sh" "$keydir" "$K8S_NAMESPACE"

# Create the `smc` namespace. This cannot be part of the YAML file as we first need to create the TLS secret,
# which would fail otherwise.
echo "Creating Kubernetes objects ..."

# Create the TLS secret for the generated keys.
kubectl -n "$K8S_NAMESPACE"  delete secret webhook-tls-certs --ignore-not-found=true
kubectl -n "$K8S_NAMESPACE" create secret tls webhook-tls-certs \
    --cert "${keydir}/webhook-tls-certs.crt" \
    --key "${keydir}/webhook-tls-certs.key"

echo "Done deploying webhook secrets"
