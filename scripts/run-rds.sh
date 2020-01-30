#!/bin/bash

rm -rf ./bin/rds

CGO_ENABLED=0 go build -v -o ./bin/rds ./cmd/rds

## GRPC_TRACE=all GRPC_VERBOSITY=DEBUG GODEBUG='http2debug=2,gctrace=1,netdns=go+1'

if [ ! -f ".env" ]; then
    echo "Create .env (see .env.example)"
    exit 1
fi

source .env

./bin/rds \
    --kubeconfig="$HOME/.kube/config" \
    --certpem="./certificates/cert.pem" \
    --keypem="./certificates/key.pem" \
    --rootcertpem="./certificates/cert.pem" \
    --azureAuthFile="$HOME/.azure/azureAuth.json" \
    --subscriptionID="$AZURE_SUBSCRIPTION" \
    --namespace="$K8S_NAMESPACE" \
    --verbosity=7
