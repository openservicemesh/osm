#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

echo -e "Deploy $PROMETHEUS_SVC cluster role"
cat <<EOF | kubectl apply -f -
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: $PROMETHEUS_SVC
rules:
- apiGroups: [""]
  resources:
  - nodes
  - nodes/proxy
  - services
  - endpoints
  - pods
  verbs: ["get", "list", "watch"]
- apiGroups:
  - extensions
  resources:
  - ingresses
  verbs: ["get", "list", "watch"]
- nonResourceURLs: ["/metrics"]
  verbs: ["get"]
---

apiVersion: v1
kind: ServiceAccount
metadata:
  name: "$PROMETHEUS_SVC-serviceaccount"
  namespace: "$K8S_NAMESPACE"
---

apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: $PROMETHEUS_SVC
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: $PROMETHEUS_SVC
subjects:
- kind: ServiceAccount
  name: "$PROMETHEUS_SVC-serviceaccount"
  namespace: "$K8S_NAMESPACE"
EOF