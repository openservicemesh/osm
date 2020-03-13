#!/usr/bin/env bash
# shellcheck disable=SC1091
set -euo pipefail

source .env

basedir="$(dirname "$0")/"
keydir="$basedir/webhook-certs"

NAME="osm"
NS="${K8S_NAMESPACE}-${NAME}"

# Generate keys into a temporary directory.
echo "Generating TLS keys ..."
"${basedir}/gen-keys.sh" "$keydir" "$NS"

# Create the `smc` namespace. This cannot be part of the YAML file as we first need to create the TLS secret,
# which would fail otherwise.
echo "Creating Kubernetes objects ..."

# Create the TLS secret for the generated keys.
kubectl -n "$NS"  delete secret webhook-tls-certs --ignore-not-found=true
kubectl -n "$NS" create secret tls webhook-tls-certs \
    --cert "${keydir}/webhook-tls-certs.crt" \
    --key "${keydir}/webhook-tls-certs.key"

echo "Done deploying webhook secrets"
