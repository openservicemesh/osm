#!/bin/bash
source .env
make docker-push-osm-controller
make docker-push-init
make build-osm
