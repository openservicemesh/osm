#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

echo -e "Disable SMI Spec policies"
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: osm-config
  namespace: $K8S_NAMESPACE

data:
  allow_all: true

EOF


kubectl delete traffictargets  -n bookstore        bookbuyer-access-bookstore-v1         --ignore-not-found
kubectl delete traffictargets  -n bookstore        bookbuyer-access-bookstore-v2         --ignore-not-found
kubectl delete traffictargets  -n bookwarehouse    bookstore-access-bookwarehouse        --ignore-not-found

kubectl delete httproutegroups -n bookstore        bookstore-service-routes              --ignore-not-found
kubectl delete httproutegroups -n bookwarehouse    bookwarehouse-service-routes          --ignore-not-found

kubectl delete trafficsplits   -n bookstore        bookstore-split                       --ignore-not-found


SMI_CRDS=$(kubectl get crds | grep smi | awk '{print $1}')

for policy in $SMI_CRDS; do
    echo -e "\n\n----- [ ${policy} ] -----"
    kubectl get "$policy" -A
done
