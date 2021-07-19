#!/bin/bash

# This script automates the collection of Envoy config during Multicluster demo/development.
# The configs are collected from Envoys in the ALPHA kube context (cluster).

kubectl config use-context alpha > /dev/null
for NAMESPACE in bookbuyer bookstore; do
    for POD in $(kubectl get pod -n "${NAMESPACE}" --no-headers | awk '{print $1}'); do
      ./osm proxy get config_dump "${POD}" --namespace="${NAMESPACE}" > "envoy__config___${NAMESPACE}___${POD}.json"
    done
done
