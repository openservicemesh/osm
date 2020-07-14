#!/bin/bash

# This script is part of the Bookstore brownfield deployment demo.
# This helper script resets the demo to what's concidered a begining state.

set -aueo pipefail

# shellcheck disable=SC1091
source .env

./demo/deploy-policies.sh
./demo/unjoin-namespaces.sh
./demo/reset-counters.sh
# ./demo/allow-all--no-policies.sh

kubectl rollout restart deployment -n osm-system zipkin
kubectl rollout restart deployment -n osm-system osm-grafana
kubectl rollout restart deployment -n osm-system osm-prometheus

rm -rf *.pcap
