#!/bin/bash

#ws docker registry push
export CTR_REGISTRY=docker.dev.ws:5000
export CTR_TAG=osmlatest28
make docker-push-osm-controller
make docker-push-init

#aws docker push
export CTR_REGISTRY=978944737929.dkr.ecr.us-west-2.amazonaws.com
make docker-push-osm-controller
make docker-push-init
