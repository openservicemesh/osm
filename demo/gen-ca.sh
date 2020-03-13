#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

DIR="./certificates"
KEY="$DIR/root-key.pem"
CRT="$DIR/root-cert.pem"

mkdir -p "$DIR"

for NAME in bookbuyer bookstore osm; do
  NS="${K8S_NAMESPACE}-${NAME}"
  openssl req -x509 -sha256 -nodes -days 365 \
        -newkey rsa:2048 \
        -subj "/CN=$(uuidgen).${NS}.azure.mesh/O=Az Mesh/C=US" \
        -keyout "$KEY"  \
        -out "$CRT"
  echo -e "Creating configmap for root cert in namespace ${NS}"
  kubectl -n "$NS" create configmap "ca-rootcertpemstore" --from-file="$CRT"
  kubectl -n "$NS" create configmap "ca-rootkeypemstore" --from-file="$KEY"
done

exit 0

./bin/cert \
    --caPEMFileOut="$CRT" \
    --caKeyPEMFileOut="$KEY" \
    --org="Azure Mesh"
