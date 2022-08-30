#!/bin/bash

# This script uninstalls the demo multicluster environment for OSM development.

# shellcheck disable=SC2128
# shellcheck disable=SC2086
cd "$(dirname ${BASH_SOURCE})"
set -e

c1=${CLUSTER1:-c1}
c2=${CLUSTER2:-c2}

# Uninstall from cluster c1
for ctx in ${c1} ${c2}; do
    kubectl config use-context "${ctx}"
    osm uninstall
    kubectl delete ns bookstore bookbuyer bookthief bookwarehouse
done
