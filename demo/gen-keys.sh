#!/bin/bash
# shellcheck disable=SC2086
: ${1?'missing key directory'}

key_dir="$1"
namespace="$2"
rm -rf "$key_dir"; mkdir -p "$key_dir"
chmod 0700 "$key_dir"

cd "$key_dir" || exit 1
# Generate the CA cert and private key
openssl req -nodes -new -x509 -keyout ca.key -out ca.crt -subj "/CN=OSM Sidecar Injection Webhook"
# Generate the private key for the webhook server
openssl genrsa -out webhook-tls-certs.key 2048
# Generate a Certificate Signing Request (CSR) for the private key, and sign it with the private key of the CA.
openssl req -new -key webhook-tls-certs.key -subj "/CN=ads.$namespace.svc" \
    | openssl x509 -req -CA ca.crt -CAkey ca.key -CAcreateserial -out webhook-tls-certs.crt
