#!/bin/bash

CGO_ENABLED=0 go build -v -o ./bin/sds ./cmd/sds

# GRPC_TRACE=all GRPC_VERBOSITY=DEBUG GODEBUG='http2debug=2,gctrace=1,netdns=go+1'

./bin/sds --keys-directory="./bin/"
