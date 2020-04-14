#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

DIR="./certificates"
KEY="$DIR/root-key.pem"
CRT="$DIR/root-cert.pem"

mkdir -p "$DIR"

./bin/cert \
    --caPEMFileOut="${CRT}" \
    --caKeyPEMFileOut="${KEY}" \
    --genca=true

echo -e "Creating configmap for root cert"

kubectl -n "$K8S_NAMESPACE" create configmap "ca-rootcertpemstore" --from-file="$CRT"
kubectl -n "$K8S_NAMESPACE" create configmap "ca-rootkeypemstore" --from-file="$KEY"
