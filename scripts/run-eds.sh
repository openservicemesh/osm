#!/bin/bash

set -aueo pipefail

rm -rf ./bin/eds

if [ ! -f ".env" ]; then
    echo "Create .env (see .env.example)"
    exit 1
fi

# shellcheck disable=SC1091
source .env

CGO_ENABLED=0 go build -v -o ./bin/eds ./cmd/eds

## GRPC_TRACE=all GRPC_VERBOSITY=DEBUG GODEBUG='http2debug=2,gctrace=1,netdns=go+1'

# We could choose a particular cipher suite like this:
# GRPC_SSL_CIPHER_SUITES=ECDHE-ECDSA-AES256-GCM-SHA384
unset GRPC_SSL_CIPHER_SUITES

# Enable gRPC debug logging
export GRPC_GO_LOG_VERBOSITY_LEVEL=99
export GRPC_GO_LOG_SEVERITY_LEVEL=info

./bin/eds \
    --kubeconfig="$HOME/.kube/config" \
    --azureAuthFile="$HOME/.azure/azureAuth.json" \
    --subscriptionID="$AZURE_SUBSCRIPTION" \
    --namespace="$K8S_NAMESPACE" \
    --verbosity=25 \
    --certpem="./certificates/cert.pem" \
    --keypem="./certificates/key.pem" \
    --rootcertpem="./certificates/root-cert.pem"
