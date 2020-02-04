#!/bin/bash

# shellcheck disable=SC1091
source .env

CONFIG_DIR="$(pwd)/demo/config"

kubectl delete configmap envoyproxy-config -n "$K8S_NAMESPACE" || true
rm -rf "${CONFIG_DIR}/*~"
rm -rf "${CONFIG_DIR}/*\\#*"
kubectl create configmap envoyproxy-config --from-file="${CONFIG_DIR}" -n "$K8S_NAMESPACE" || true
