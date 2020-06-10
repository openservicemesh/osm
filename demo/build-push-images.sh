#!/bin/bash

set -aueo pipefail

make docker-push-osm-controller

make docker-push-init
make docker-push-bookbuyer
make docker-push-bookthief
make docker-push-bookstore
make docker-push-bookwarehouse
