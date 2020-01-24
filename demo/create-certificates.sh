#!/bin/bash

set -aueo pipefail

source .env

ROOT_KEY="root.key"
ROOT_CERT="root.crt"
CERT_KEY="cert.key"

CERT_SERVER="server.crt"
CERT_CLIENT="server.crt"

CERT_DIR="certificates/"
CERT_CONF="$(pwd)/demo/certs.conf"

echo "Creating certificates in directory: $CERT_DIR"

rm -rf "$(pwd)/$CERT_DIR"
mkdir -p "$(pwd)/$CERT_DIR"
pushd "$(pwd)/$CERT_DIR"

# Create Root Key
openssl genrsa -out $ROOT_KEY 4096


# Create and self sign the Root Certificate
openssl req \
        -config "$CERT_CONF" \
        -x509 \
        -new \
        -nodes \
        -key $ROOT_KEY \
        -sha256 \
        -days 1024 \
        -out $ROOT_CERT

# Create the certificate key
openssl genrsa -out $CERT_KEY 2048

declare -a domains
domains=(\
    "bookbuyer.$K8S_NAMESPACE.svc.cluster.local" \
    "bookstore.$K8S_NAMESPACE.svc.cluster.local" \
    "bookstore-1.$K8S_NAMESPACE.svc.cluster.local" \
    "bookstore-2.$K8S_NAMESPACE.svc.cluster.local" )

for DOMAIN in ${domains[@]}; do

    CSR="${DOMAIN}.csr"
    CERT="${DOMAIN}.crt"

    # Create the signing (csr)
    openssl req \
            -new \
            -sha256 \
            -key $CERT_KEY \
            -subj "/C=US/ST=CA/O=Mesh, Inc./CN=${DOMAIN}" \
            -out $CSR


    # Generate the certificate using the mydomain csr and key along with the CA Root key
    openssl x509 \
            -req \
            -in $CSR \
            -CA $ROOT_CERT \
            -CAkey $ROOT_KEY \
            -CAcreateserial  \
            -out $CERT \
            -days 500 \
            -sha256


    # Verify the certificate's content
    openssl x509 \
            -in $CERT \
            -text \
            -noout
done
