#!/bin/bash

rm -rf ./bin/eds

CGO_ENABLED=0 go build -v -o ./bin/eds ./cmd/eds

## GRPC_TRACE=all GRPC_VERBOSITY=DEBUG GODEBUG='http2debug=2,gctrace=1,netdns=go+1'

if [ ! -f ".env" ]; then
    echo "Create .env (see .env.example)"
    exit 1
fi

source .env

./bin/eds \
    --kubeconfig="$HOME/.kube/config" \
    --resource-group="$AZURE_RESOURCE_GROUP" \
    --azureAuthFile="$HOME/.azure/azureAuth.json" \
    --subscriptionID="$AZURE_SUBSCRIPTION" \
    --namespace="$K8S_NAMESPACE" \
    --verbosity=7
