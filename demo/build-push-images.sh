#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

make docker-push-ads

make docker-push-init
if [[ "$IS_GITHUB" != "true" ]]; then
    make docker-push-envoyproxy
fi
make docker-push-bookbuyer
make docker-push-bookstore
