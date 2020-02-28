#!/bin/bash

set -aueo pipefail

DIR="./certificates"
KEY="$DIR/root-key.pem"
CRT="$DIR/root-cert.pem"

mkdir -p "$DIR"

openssl req -x509 -sha256 -nodes -days 365 \
        -newkey rsa:2048 \
        -subj "/CN=$(uuidgen).azure.mesh/O=Az Mesh/C=US" \
        -keyout "$KEY"  \
        -out "$CRT"

kubectl -n "$K8S_NAMESPACE" create configmap "ca-rootcertpemstore" --from-file="$CRT"
kubectl -n "$K8S_NAMESPACE" create configmap "ca-rootkeypemstore" --from-file="$KEY"

exit 0

./bin/cert \
    --caPEMFileOut="$CRT" \
    --caKeyPEMFileOut="$KEY" \
    --org="Azure Mesh"
