#!/bin/bash
# shellcheck disable=SC2086
: ${1?'missing key directory'}

key_dir="$1"
rm -rf "$key_dir"; mkdir -p "$key_dir"
chmod 0700 "$key_dir"

cd "$key_dir" || exit 1
# Generate the CA cert and private key
openssl req -nodes -new -x509 -keyout ca.key -out ca.crt -subj "/CN=SMC Sidecar Injection Webhook"
# Generate the private key for the webhook server
openssl genrsa -out tls-webhook-server.key 2048
# Generate a Certificate Signing Request (CSR) for the private key, and sign it with the private key of the CA.
openssl req -new -key tls-webhook-server.key -subj "/CN=ads.smc.svc" \
    | openssl x509 -req -CA ca.crt -CAkey ca.key -CAcreateserial -out tls-webhook-server.crt
