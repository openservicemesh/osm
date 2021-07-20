#!/bin/bash

# This script automates the collection of Envoy config during Multicluster demo/development.
# The configs are collected from Envoys in the ALPHA kube context (cluster).

TMP_PID_FILE=$(mktemp)

PORT=15000

kubectl config use-context alpha > /dev/null
for NAMESPACE in bookbuyer bookstore; do
    for POD in $(kubectl get pod -n "${NAMESPACE}" --no-headers | awk '{print $1}'); do
      ( kubectl port-forward --namespace="${NAMESPACE}" "${POD}" "${PORT}" & echo $! >&3 ) 3> "${TMP_PID_FILE}"
      sleep 1
      curl "http://localhost:${PORT}/config_dump?include_eds"  >  "envoy__config___${NAMESPACE}___${POD}.json"
      kill -9 $(<"${TMP_PID_FILE}")
      rm -rf "${TMP_PID_FILE}"
      # ./osm proxy get config_dump "${POD}" --namespace="${NAMESPACE}" > "envoy__config___${NAMESPACE}___${POD}.json"
    done
done
