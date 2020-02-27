#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

NAME=${1:-unknown}

echo -e "Generate certificates for ${NAME}"
mkdir -p "./certificates/$NAME/"

./bin/cert --host="$NAME.azure.mesh" \
           --caPEMFileIn="./certificates/root-cert.pem" \
           --caKeyPEMFileIn="./certificates/root-key.pem" \
           --keyout "./certificates/$NAME/key.pem" \
           --out "./certificates/$NAME/cert.pem"

echo -e "Add secrets"
kubectl -n "$K8S_NAMESPACE" create configmap "ca-rootcertpemstore-${NAME}" --from-file="./certificates/root-cert.pem"
kubectl -n "$K8S_NAMESPACE" create configmap "ca-certpemstore-${NAME}" --from-file="./certificates/$NAME/cert.pem"
kubectl -n "$K8S_NAMESPACE" create configmap "ca-keypemstore-${NAME}" --from-file="./certificates/$NAME/key.pem"
