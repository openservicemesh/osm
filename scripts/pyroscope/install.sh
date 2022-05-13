#!/bin/bash

# RUN THIS SCRIPT FROM THE ROOT OF THE REPOSITORY

# shellcheck disable=SC1091
source .env

kubectl patch meshconfig osm-mesh-config -n "$K8S_NAMESPACE" -p '{"spec":{"observability":{"enableDebugServer":true}}}' --type=merge

kubectl annotate -n "$K8S_NAMESPACE" svc osm-controller "pyroscope.io/scrape=true" "pyroscope.io/application-name=osm-controller" "pyroscope.io/spy-name=gospy" "pyroscope.io/profile-cpu-enabled=true" "pyroscope.io/profile-mem-enabled=true" "pyroscope.io/port=9092"

helm install prof pyroscope-io/pyroscope -f scripts/pyroscope/values.yaml
