#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

# Deploy apps in the order of their dependencies to avoid initial timing errors
# in osm-controller logs. Server apps are deployed before client apps.

# Deploy bookwarehouse
./demo/deploy-bookwarehouse.sh
# Deploy bookstore versions
./demo/deploy-bookstore.sh "v1"
./demo/deploy-bookstore.sh "v2"
# Deploy bookbuyer
./demo/deploy-bookbuyer.sh
# Deploy bookthief
./demo/deploy-bookthief.sh
