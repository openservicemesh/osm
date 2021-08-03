#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env
DEPLOY_ON_OPENSHIFT="${DEPLOY_ON_OPENSHIFT:-false}"
USE_PRIVATE_REGISTRY="${USE_PRIVATE_REGISTRY:-true}"

kubectl apply -f docs/example/manifests/apps/mysql.yaml -n "$BOOKWAREHOUSE_NAMESPACE"

kubectl apply -f - <<EOF
kind: TrafficTarget
apiVersion: access.smi-spec.io/v1alpha3
metadata:
  name: mysql
  namespace: $BOOKWAREHOUSE_NAMESPACE
spec:
  destination:
    kind: ServiceAccount
    name: mysql
    namespace: $BOOKWAREHOUSE_NAMESPACE
  rules:
  - kind: TCPRoute
    name: tcp-route
  sources:
  - kind: ServiceAccount
    name: bookwarehouse
    namespace: $BOOKWAREHOUSE_NAMESPACE
---
apiVersion: specs.smi-spec.io/v1alpha4
kind: TCPRoute
metadata:
  name: tcp-route
  namespace: $BOOKWAREHOUSE_NAMESPACE
spec:
  matches:
    ports:
    - 3306
EOF

if [ "$DEPLOY_ON_OPENSHIFT" = true ] ; then
    oc adm policy add-scc-to-user privileged -z "mysql" -n "$BOOKWAREHOUSE_NAMESPACE"
    if [ "$USE_PRIVATE_REGISTRY" = true ]; then
        oc secrets link "mysql" "$CTR_REGISTRY_CREDS_NAME" --for=pull -n "$BOOKWAREHOUSE_NAMESPACE"
    fi
fi

