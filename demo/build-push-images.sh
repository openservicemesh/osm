#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

make docker-push-ads

make docker-push-init
make docker-push-envoyproxy
make docker-push-bookbuyer
make docker-push-bookstore
