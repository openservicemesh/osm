#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

promethuesConfig=$(<./demo/metrics/prometheus/prometheus-config.txt)

echo -e "Deploy $PROMETHEUS_SVC config map"
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: "$PROMETHEUS_SVC-server-conf"
  labels:
    name: "$PROMETHEUS_SVC-server-conf"
  namespace: "$K8S_NAMESPACE"
$promethuesConfig  
EOF