#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

echo -e "Delete old secrets"
kubectl -n "$K8S_NAMESPACE" delete configmap ca-certpemstore ca-keypemstore || true
echo -e "Add secrets"
kubectl -n "$K8S_NAMESPACE" create configmap ca-certpemstore --from-file=./bin/cert.pem
kubectl -n "$K8S_NAMESPACE" create configmap ca-keypemstore --from-file=./bin/key.pem
