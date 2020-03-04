#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env
IS_GITHUB="${IS_GITHUB:-default false}"

make docker-push-ads

make docker-push-init
if [[ "$IS_GITHUB" != "true" ]]; then
    make docker-push-envoyproxy
fi
make docker-push-bookbuyer
make docker-push-bookstore
