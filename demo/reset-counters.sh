#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

# This script assumes that port forwarding has already been established.
# See ./scripts/port-forward-all.sh to enable fort forwarding for Bookstore demo.
curl -I -X GET http://localhost:8080/reset &
curl -I -X GET http://localhost:8081/reset &
curl -I -X GET http://localhost:8082/reset &
curl -I -X GET http://localhost:8083/reset &

# Restart these pods to reset their data stores.
kubectl rollout restart deployment -n osm-system jaeger &
kubectl rollout restart deployment -n osm-system osm-grafana &
kubectl rollout restart deployment -n osm-system osm-prometheus &

wait
