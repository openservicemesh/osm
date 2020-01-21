#!/bin/bash

set -aueo pipefail

source .env

make docker-push-cds
make docker-push-lds
make docker-push-eds
make docker-push-sds
make docker-push-rds

make docker-push-init
make docker-push-bookbuyer
make docker-push-bookstore
