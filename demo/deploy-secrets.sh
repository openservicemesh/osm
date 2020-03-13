#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

NAME=${1:-unknown}
NS=${2:-default}

echo -e "Delete old secrets: ca-certpemstore-${NAME}, ca-keypemstore-${NAME}"
kubectl -n "$NS" \
        delete configmap \
        "ca-certpemstore-${NAME}" \
        "ca-keypemstore-${NAME}" || true

echo -e "Generate certificates for ${NAME}"
mkdir -p "./certificates/$NAME/"

./bin/cert --host="$NAME.$NS.azure.mesh" \
           --caPEMFileIn="./certificates/root-cert.pem" \
           --caKeyPEMFileIn="./certificates/root-key.pem" \
           --keyout "./certificates/$NAME/key.pem" \
           --out "./certificates/$NAME/cert.pem"

echo -e "Add secrets for ${NAME}"
kubectl -n "$NS" create configmap "ca-certpemstore-${NAME}" --from-file="./certificates/$NAME/cert.pem"
kubectl -n "$NS" create configmap "ca-keypemstore-${NAME}" --from-file="./certificates/$NAME/key.pem"
