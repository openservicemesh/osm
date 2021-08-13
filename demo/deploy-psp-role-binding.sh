#!/bin/bash

# shellcheck disable=SC1091
source .env

echo -e "Deploy ClusterRoleBinding for ServiceAccount $1 to ClusterRole osm-demo"
kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: osm-demo-$1
subjects:
  - kind: ServiceAccount
    name: "$1"
    namespace: "$2"
roleRef:
  kind: ClusterRole
  name: osm-demo
  apiGroup: rbac.authorization.k8s.io
EOF