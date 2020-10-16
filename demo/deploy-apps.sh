#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

DEPLOY_WITH_SAME_SA="${DEPLOY_WITH_SAME_SA:-false}"

# Deploy apps in the order of their dependencies to avoid initial timing errors
# in osm-controller logs. Server apps are deployed before client apps.

# Deploy bookwarehouse
./demo/deploy-bookwarehouse.sh

# Deploy bookstore versions
if [ "$DEPLOY_WITH_SAME_SA" = "true" ]; then
    ./demo/deploy-bookstore-with-same-sa.sh "v1"
    ./demo/deploy-bookstore-with-same-sa.sh "v2"
else
    ./demo/deploy-bookstore.sh "v1"
    ./demo/deploy-bookstore.sh "v2"
fi

# Deploy bookbuyer
./demo/deploy-bookbuyer.sh

# Deploy bookthief
./demo/deploy-bookthief.sh
