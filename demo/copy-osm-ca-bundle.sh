#!/bin/bash

set -auexo pipefail

# shellcheck disable=SC1091
source .env

ALPHA_CLUSTER="${ALPHA_CLUSTER:-alpha}"
BETA_CLUSTER="${BETA_CLUSTER:-beta}"

kubectl config use-context "$ALPHA_CLUSTER"
kubectl get secret osm-ca-bundle -n osm-system -o yaml > /tmp/ca-bundle.yaml


kubectl config use-context "$BETA_CLUSTER"

kubectl apply -f /tmp/ca-bundle.yaml
rm -f /tmp/ca-bundle.yaml
