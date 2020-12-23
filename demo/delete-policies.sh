#!/bin/bash



# This script removes all SMI Spec policies and switches OSM to permissive traffic mode.
# This is a helper script part of brownfield OSM demo.



set -aueo pipefail

# shellcheck disable=SC1091
source .env



K8S_NAMESPACE="${K8S_NAMESPACE:-osm-system}"
BOOKBUYER_NAMESPACE="${BOOKBUYER_NAMESPACE:-bookbuyer}"
BOOKSTORE_NAMESPACE="${BOOKSTORE_NAMESPACE:-bookstore}"
BOOKTHIEF_NAMESPACE="${BOOKTHIEF_NAMESPACE:-bookthief}"
BOOKWAREHOUSE_NAMESPACE="${BOOKWAREHOUSE_NAMESPACE:-bookwarehouse}"


echo -e "Enable Permissive Traffic Policy Mode & Disable SMI Spec policies"
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap

metadata:
  name: osm-config
  namespace: $K8S_NAMESPACE

data:
  permissive_traffic_policy_mode: "true"
  egress: "false"
  prometheus_scraping: "false"
  tracing_enable: "false"

EOF


kubectl delete traffictargets  -n "$BOOKSTORE_NAMESPACE"     bookbuyer-access-bookstore-v1  --ignore-not-found
kubectl delete traffictargets  -n "$BOOKSTORE_NAMESPACE"     bookbuyer-access-bookstore-v2  --ignore-not-found
kubectl delete traffictargets  -n "$BOOKWAREHOUSE_NAMESPACE" bookstore-access-bookwarehouse --ignore-not-found

kubectl delete httproutegroups -n "$BOOKSTORE_NAMESPACE"     bookstore-service-routes       --ignore-not-found
kubectl delete httproutegroups -n "$BOOKWAREHOUSE_NAMESPACE" bookwarehouse-service-routes   --ignore-not-found

kubectl delete trafficsplits   -n "$BOOKSTORE_NAMESPACE"     bookstore-split                --ignore-not-found


SMI_CRDS=$(kubectl get crds | grep smi | awk '{print $1}')

for policy in $SMI_CRDS; do
    echo -e "\n\n----- [ ${policy} ] -----"
    kubectl get "$policy" -A
done
