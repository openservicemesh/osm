#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

make build-osm
make docker-build-demo

MESH_NAME="${MESH_NAME:-osm}"
CTR_TAG="${CTR_TAG:-latest-main}"
CTR_REGISTRY="${CTR_REGISTRY:-localhost:5000}"
CTR_REGISTRY_CREDS_NAME="${CTR_REGISTRY_CREDS_NAME:-acr-creds}"

./demo/configure-app-namespaces.sh
./demo/deploy-apps.sh
./demo/deploy-traffic-split.sh
./demo/deploy-traffic-specs.sh
./demo/deploy-traffic-target.sh
