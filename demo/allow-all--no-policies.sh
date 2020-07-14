#!/bin/bash

# This script is a helper, which deletes SMI policies and also switches to Allow All mode.

set -aueo pipefail

# shellcheck disable=SC1091
source .env

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap

metadata:
  name: osm-config
  namespace: $K8S_NAMESPACE

data:

  permissive_traffic_policy_mode: true

EOF

kubectl delete trafficsplit -n bookstore bookstore-split

kubectl delete traffictarget -n bookstore       bookbuyer-access-bookstore-v1
kubectl delete traffictarget -n bookstore       bookbuyer-access-bookstore-v2
kubectl delete traffictarget -n bookwarehouse   bookstore-access-bookwarehouse
