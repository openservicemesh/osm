#!/bin/bash

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
