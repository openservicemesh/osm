#!/bin/bash

set -aueo pipefail

source .env

echo -e "Add secrets"
kubectl -n smc delete configmap ca-certpemstore ca-keypemstore || true
kubectl -n smc create configmap ca-certpemstore --from-file=./bin/cert.pem
kubectl -n smc create configmap ca-keypemstore --from-file=./bin/key.pem
