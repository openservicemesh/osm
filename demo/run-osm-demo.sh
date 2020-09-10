#!/bin/bash

set -aueo pipefail

if [ ! -f .env ]; then
    echo -e "\nThere is no .env file in the root of this repository."
    echo -e "Copy the values from .env.example into .env."
    echo -e "Modify the values in .env to match your setup.\n"
    echo -e "    cat .env.example > .env\n\n"
    exit 1
fi

# shellcheck disable=SC1091
source .env

# Set meaningful defaults for env vars we expect from .env
CI="${CI:-false}"  # This is set to true by Github Actions
MESH_NAME="${MESH_NAME:-osm}"
K8S_NAMESPACE="${K8S_NAMESPACE:-osm-system}"
BOOKBUYER_NAMESPACE="${BOOKBUYER_NAMESPACE:-bookbuyer}"
BOOKSTORE_NAMESPACE="${BOOKSTORE_NAMESPACE:-bookstore}"
BOOKTHIEF_NAMESPACE="${BOOKTHIEF_NAMESPACE:-bookthief}"
BOOKWAREHOUSE_NAMESPACE="${BOOKWAREHOUSE_NAMESPACE:-bookwarehouse}"
CERT_MANAGER="${CERT_MANAGER:-tresor}"
CTR_REGISTRY="${CTR_REGISTRY:-localhost:5000}"
CTR_REGISTRY_CREDS_NAME="${CTR_REGISTRY_CREDS_NAME:-acr-creds}"
DEPLOY_TRAFFIC_SPLIT="${DEPLOY_TRAFFIC_SPLIT:-true}"
CTR_TAG="${CTR_TAG:-$(git rev-parse HEAD)}"
IMAGE_PULL_POLICY="${IMAGE_PULL_POLICY:-Always}"
ENABLE_EGRESS="${ENABLE_EGRESS:-false}"
ENABLE_GRAFANA="${ENABLE_GRAFANA:-false}"
MESH_CIDR=$(./scripts/get_mesh_cidr.sh)
DEPLOY_WITH_SAME_SA="${DEPLOY_WITH_SAME_SA:-false}"
ENVOY_LOG_LEVEL="${ENVOY_LOG_LEVEL:-debug}"

# For any additional installation arguments. Used heavily in CI.
optionalInstallArgs=$*

exit_error() {
    error="$1"
    echo "$error"
    exit 1
}

wait_for_osm_pods() {
  # Wait for OSM pods to be ready before deploying the apps.
  pods=$(kubectl get pods -n "$K8S_NAMESPACE" -o name | sed 's/^pod\///')
  if [ -n "$pods" ]; then
    for pod in $pods; do
      wait_for_pod_ready "$pod"
    done
  else
    exit_error "No Pods found in namespace $K8S_NAMESPACE"
  fi
}

wait_for_pod_ready() {
    max=15
    pod_name=$1

    for x in $(seq 1 $max); do
        pod_ready="$(kubectl get pods -n "$K8S_NAMESPACE" "${pod_name}" -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}')"
        if [ "$pod_ready" == "True" ]; then
            echo "[${x}] Pod ready ${pod_name}"
            return
        fi

        pod_status="$(kubectl get pods -n "$K8S_NAMESPACE" "${pod_name}" -o 'jsonpath={..status.phase}')"
        echo "[${x}] Pod status is ${pod_status}; waiting for pod ${pod_name} to be Ready" && sleep 5
    done

    pod_status="$(kubectl get pods -n "$K8S_NAMESPACE" "${pod_name}" -o 'jsonpath={..status.phase}')"
    exit_error "Pod ${pod_name} status is ${pod_status} -- still not Ready"
}

make build-osm

# cleanup stale resources from previous runs
bin/osm mesh uninstall -f --mesh-name "$MESH_NAME" --namespace "$K8S_NAMESPACE"
./demo/clean-kubernetes.sh

# The demo uses osm's namespace as defined by environment variables, K8S_NAMESPACE
# to house the control plane components.
kubectl create namespace "$K8S_NAMESPACE"

echo "Certificate Manager in use: $CERT_MANAGER"
if [ "$CERT_MANAGER" = "vault" ]; then
    echo "Installing Hashi Vault"
    ./demo/deploy-vault.sh
fi

if [ "$CERT_MANAGER" = "cert-manager" ]; then
    echo "Installing cert-manager"
    ./demo/deploy-cert-manager.sh
fi

make docker-push
./scripts/create-container-registry-creds.sh "$K8S_NAMESPACE"

# Deploys Xds and Prometheus
echo "Certificate Manager in use: $CERT_MANAGER"
if [ "$CERT_MANAGER" = "vault" ]; then
  # shellcheck disable=SC2086
  bin/osm install \
      --namespace "$K8S_NAMESPACE" \
      --mesh-name "$MESH_NAME" \
      --certificate-manager="$CERT_MANAGER" \
      --vault-host="$VAULT_HOST" \
      --vault-token="$VAULT_TOKEN" \
      --vault-protocol="$VAULT_PROTOCOL" \
      --container-registry "$CTR_REGISTRY" \
      --container-registry-secret "$CTR_REGISTRY_CREDS_NAME" \
      --osm-image-tag "$CTR_TAG" \
      --osm-image-pull-policy "$IMAGE_PULL_POLICY" \
      --enable-debug-server \
      --enable-egress="$ENABLE_EGRESS" \
      --enable-grafana="$ENABLE_GRAFANA" \
      --mesh-cidr "$MESH_CIDR" \
      --envoy-log-level "$ENVOY_LOG_LEVEL" \
      $optionalInstallArgs
else
  # shellcheck disable=SC2086
  bin/osm install \
      --namespace "$K8S_NAMESPACE" \
      --mesh-name "$MESH_NAME" \
      --certificate-manager="$CERT_MANAGER" \
      --container-registry "$CTR_REGISTRY" \
      --container-registry-secret "$CTR_REGISTRY_CREDS_NAME" \
      --osm-image-tag "$CTR_TAG" \
      --osm-image-pull-policy "$IMAGE_PULL_POLICY" \
      --enable-debug-server \
      --enable-egress="$ENABLE_EGRESS" \
      --enable-grafana="$ENABLE_GRAFANA" \
      --mesh-cidr "$MESH_CIDR" \
      --envoy-log-level "$ENVOY_LOG_LEVEL" \
      $optionalInstallArgs
fi

wait_for_osm_pods

./demo/configure-app-namespaces.sh
bin/osm namespace add --mesh-name "$MESH_NAME" "$BOOKWAREHOUSE_NAMESPACE" "$BOOKBUYER_NAMESPACE" "$BOOKSTORE_NAMESPACE" "$BOOKTHIEF_NAMESPACE" --enable-sidecar-injection
./demo/deploy-apps.sh

# Apply SMI policies
if [ "$DEPLOY_TRAFFIC_SPLIT" = "true" ]; then
    ./demo/deploy-traffic-split.sh
fi

./demo/deploy-traffic-specs.sh

if [ "$DEPLOY_WITH_SAME_SA" = "true" ]; then
    ./demo/deploy-traffic-target-with-same-sa.sh
else
    ./demo/deploy-traffic-target.sh
fi

if [[ "$CI" != "true" ]]; then
    watch -n5 "printf \"Namespace ${K8S_NAMESPACE}:\n\"; kubectl get pods -n ${K8S_NAMESPACE} -o wide; printf \"\n\n\"; printf \"Namespace ${BOOKBUYER_NAMESPACE}:\n\"; kubectl get pods -n ${BOOKBUYER_NAMESPACE} -o wide; printf \"\n\n\"; printf \"Namespace ${BOOKSTORE_NAMESPACE}:\n\"; kubectl get pods -n ${BOOKSTORE_NAMESPACE} -o wide; printf \"\n\n\"; printf \"Namespace ${BOOKTHIEF_NAMESPACE}:\n\"; kubectl get pods -n ${BOOKTHIEF_NAMESPACE} -o wide; printf \"\n\n\"; printf \"Namespace ${BOOKWAREHOUSE_NAMESPACE}:\n\"; kubectl get pods -n ${BOOKWAREHOUSE_NAMESPACE} -o wide"
fi
