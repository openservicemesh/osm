#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

# Deploy bookstore
./demo/deploy-bookstore.sh "v1"
./demo/deploy-bookstore.sh "v2"
# Deploy bookbuyer
./demo/deploy-bookbuyer.sh
# Deploy bookthief
./demo/deploy-bookthief.sh

./demo/deploy-bookwarehouse.sh
