#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

kubectl delete mutatingwebhookconfiguration ads --ignore-not-found=true
kubectl delete namespace "$K8S_NAMESPACE" || true
kubectl create namespace "$K8S_NAMESPACE" || true
kubectl delete clusterrole osm-xds || true
kubectl delete clusterrolebinding osm-xds || true
