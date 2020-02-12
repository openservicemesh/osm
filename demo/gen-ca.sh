#!/bin/bash

set -aueo pipefail

DIR="./certs"

mkdir -p "$DIR"

openssl req -x509 -sha256 -nodes -days 365 \
        -newkey rsa:2048 \
        -subj "/CN=$(uuidgen).azure.mesh/O=Az Mesh/C=US" \
        -keyout $DIR/root-key.pem \
        -out $DIR/root-cert.pem


exit 0

./bin/cert \
    --caPEMFileOut="$DIR/root-cert.pem" \
    --caKeyPEMFileOut="$DIR/root-key.pem" \
    --org="Azure Mesh"
