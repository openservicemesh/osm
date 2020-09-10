#!/bin/bash
export CTR_REGISTRY=docker.dev.ws:5000
export CTR_TAG=osmlatest6
make docker-push-osm-controller
make docker-push-init
#make build-osm
