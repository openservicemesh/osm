#!/bin/bash

set -aueo pipefail

DIR="./certificates"
KEY="$DIR/root-key.pem"
CRT="$DIR/root-cert.pem"

mkdir -p "$DIR"

./bin/cert \
    --caPEMFileOut="$CRT" \
    --caKeyPEMFileOut="$KEY" \
    --org="Azure Mesh"
