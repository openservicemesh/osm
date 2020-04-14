#!/bin/bash

set -auexo pipefail

# shellcheck disable=SC1091
source .env

# Deploy bookstore
./demo/deploy-bookstore.sh "bookstore-1"
./demo/deploy-bookstore.sh "bookstore-2"
# Deploy bookbuyer
./demo/deploy-bookbuyer.sh
# Deploy bookthief
./demo/deploy-bookthief.sh

# Apply SMI policies
./demo/deploy-traffic-split.sh
./demo/deploy-traffic-spec.sh
./demo/deploy-traffic-target.sh
