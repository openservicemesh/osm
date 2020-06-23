#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

echo -e "Deploy $SVC demo service"
cat <<EOF | kubectl apply -f -
apiVersion: osm.k8s.io/v1
kind: OSMConfig
metadata:
  name: osm-config
  namespace: $K8S_NAMESPACE
spec:
  logVerbosity: trace
EOF
