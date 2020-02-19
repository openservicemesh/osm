#!/bin/bash

make build-cert

./demo/gen-ca.sh

for NAME in ads bookbuyer bookstore; do
    echo -e "Generate certificates for ${NAME}"
    mkdir -p "./certificates/$NAME/"

    ./bin/cert --host="$NAME.azure.mesh" \
               --caPEMFileIn="./certificates/root-cert.pem" \
               --caKeyPEMFileIn="./certificates/root-key.pem" \
               --keyout "./certificates/$NAME/key.pem" \
               --out "./certificates/$NAME/cert.pem"
done
