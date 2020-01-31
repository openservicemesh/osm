#!/bin/bash

set -aueo pipefail

rm -rf ./bin/lds

CGO_ENABLED=0 go build -v -o ./bin/lds ./cmd/lds

# GRPC_TRACE=all GRPC_VERBOSITY=DEBUG GODEBUG='http2debug=2,gctrace=1,netdns=go+1'

# We could choose a particular cipher suite like this:
# GRPC_SSL_CIPHER_SUITES=ECDHE-ELDSA-AES256-GCM-SHA384
unset GRPC_SSL_CIPHER_SUITES

# Enable gRPC debug logging
export GRPC_GO_LOG_VERBOSITY_LEVEL=99
export GRPC_GO_LOG_SEVERITY_LEVEL=info

./bin/lds \
    --kubeconfig="$HOME/.kube/config" \
    --certpem="./certificates/cert.pem" \
    --keypem="./certificates/key.pem" \
    --rootcertpem="./certificates/cert.pem" \
    --verbosity=25
