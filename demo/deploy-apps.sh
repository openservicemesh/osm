#!/bin/bash

set -auexo pipefail

# shellcheck disable=SC1091

# create and label app namespaces, create container registry secrets
./demo/configure-app-namespaces.sh

# Deploy bookstore
./demo/deploy-bookstore.sh "v1"
./demo/deploy-bookstore.sh "v2"
# Deploy bookbuyer
./demo/deploy-bookbuyer.sh
# Deploy bookthief
./demo/deploy-bookthief.sh
# Deploy bookwarehouse
./demo/deploy-bookwarehouse.sh
