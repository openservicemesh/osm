#!/bin/bash

# This is a helper script for quickly updating the osm-config ConfigMap
# during demos and local testing.

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
  config_version: "111"
  permissive_traffic_policy_mode: "true"

EOF
