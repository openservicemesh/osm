#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

OSM_NS="${K8S_NAMESPACE}-osm"
POD="$(kubectl get pods -n "$OSM_NS" --selector app=ads --no-headers | awk '{print $1}' | head -n1)"

kubectl logs "${POD}" -n "$OSM_NS" -f
