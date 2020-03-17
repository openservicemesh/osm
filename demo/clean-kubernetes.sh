#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

WEBHOOK_NAME="$K8S_NAMESPACE-ads-webhook"

kubectl delete mutatingwebhookconfiguration "$WEBHOOK_NAME" --ignore-not-found=true
kubectl delete namespace "$K8S_NAMESPACE" || true
kubectl delete clusterrole osm-xds || true
kubectl delete clusterrolebinding osm-xds || true
