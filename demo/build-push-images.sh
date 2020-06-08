#!/bin/bash

set -aueo pipefail

make docker-push-ads

make docker-push-init
make docker-push-bookbuyer
make docker-push-bookthief
make docker-push-bookstore
make docker-push-bookwarehouse
