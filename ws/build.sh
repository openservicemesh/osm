#!/bin/bash
source .env
#make build-osm
#make build
make docker-push-osm-controller
make docker-push-init
#make build-osm
