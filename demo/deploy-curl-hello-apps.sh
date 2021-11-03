#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

# Deploy hello server
./demo/deploy-hello.sh

# Deploy curl
./demo/deploy-curl.sh
