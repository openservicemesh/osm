#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env
IS_GITHUB="${IS_GITHUB:-false}"
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
    max=12
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

# Check for required environment variables
if [ -z "$K8S_NAMESPACE" ]; then
    exit_error "Missing K8S_NAMESPACE env variable"
fi
if [ -z "$BOOKBUYER_NAMESPACE" ]; then
    exit_error "Missing BOOKBUYER_NAMESPACE env variable"
fi
if [ -z "$BOOKSTORE_NAMESPACE" ]; then
    exit_error "Missing BOOKSTORE_NAMESPACE env variable"
fi
if [ -z "$BOOKTHIEF_NAMESPACE" ]; then
    exit_error "Missing BOOKTHIEF_NAMESPACE env variable"
fi
if [ -z "$BOOKWAREHOUSE_NAMESPACE" ]; then
    exit_error "Missing BOOKWAREHOUSE_NAMESPACE env variable"
fi
if [ -z "$CERT_MANAGER" ]; then
    exit_error "Missing CERT_MANAGER env variable"
fi
if [ -z "$CTR_REGISTRY" ]; then
    exit_error "Missing CTR_REGISTRY env variable"
fi
if [ -z "$CTR_REGISTRY_CREDS_NAME" ]; then
    exit_error "Missing CTR_REGISTRY_CREDS_NAME env variable"
fi
if [ -z "$CTR_TAG" ]; then
    exit_error "Missing CTR_TAG env variable"
fi

make build-osm

if [[ "$IS_GITHUB" != "true" ]]; then
    # In Github CI we always use a new namespace - so this is not necessary
    bin/osm mesh delete "$MESH_NAME" --namespace "$K8S_NAMESPACE" || true
    ./demo/clean-kubernetes.sh
else
    bin/osm mesh delete "$MESH_NAME" --namespace "$K8S_NAMESPACE" || true
fi

# Run pre-install checks to make sure OSM can be installed in the current kubectl context.
bin/osm check --pre-install --namespace "$K8S_NAMESPACE"

# The demo uses osm's namespace as defined by environment variables, K8S_NAMESPACE
# to house the control plane components.
kubectl create namespace "$K8S_NAMESPACE"

echo "Certificate Manager in use: $CERT_MANAGER"
if [ "$CERT_MANAGER" = "vault" ]; then
    echo "Installing Hashi Vault"
    ./demo/deploy-vault.sh
fi

if [[ "$IS_GITHUB" != "true" ]]; then
    # For Github CI we achieve these at a different time or different script
    # See .github/workflows/main.yml
    ./demo/build-push-images.sh
    ./demo/create-container-registry-creds.sh
else
    # This script is specifically for CI
    ./ci/create-osm-container-registry-creds.sh
fi

for ns in "$BOOKWAREHOUSE_NAMESPACE" "$BOOKBUYER_NAMESPACE" "$BOOKSTORE_NAMESPACE" "$BOOKTHIEF_NAMESPACE"; do
    kubectl create namespace "$ns"
    bin/osm namespace add --mesh-name "$MESH_NAME" "$ns"
done

# Deploys Xds and Prometheus
echo "Certificate Manager in use: $CERT_MANAGER"
if [ "$CERT_MANAGER" = "vault" ]; then
  # shellcheck disable=SC2086
  bin/osm install \
      --namespace "$K8S_NAMESPACE" \
      --mesh-name "$MESH_NAME" \
      --cert-manager="$CERT_MANAGER" \
      --vault-host="$VAULT_HOST" \
      --vault-token="$VAULT_TOKEN" \
      --vault-protocol="$VAULT_PROTOCOL" \
      --container-registry "$CTR_REGISTRY" \
      --container-registry-secret "$CTR_REGISTRY_CREDS_NAME" \
      --osm-image-tag "$CTR_TAG" \
      --enable-debug-server \
      $optionalInstallArgs
else
  # shellcheck disable=SC2086
  bin/osm install \
      --namespace "$K8S_NAMESPACE" \
      --mesh-name "$MESH_NAME" \
      --container-registry "$CTR_REGISTRY" \
      --container-registry-secret "$CTR_REGISTRY_CREDS_NAME" \
      --osm-image-tag "$CTR_TAG" \
      --enable-debug-server \
      $optionalInstallArgs
fi

wait_for_osm_pods

./demo/deploy-apps.sh

# Apply SMI policies
./demo/deploy-traffic-split.sh
./demo/deploy-traffic-spec.sh
./demo/deploy-traffic-target.sh

if [[ "$IS_GITHUB" != "true" ]]; then
    watch -n5 "printf \"Namespace ${K8S_NAMESPACE}:\n\"; kubectl get pods -n ${K8S_NAMESPACE} -o wide; printf \"\n\n\"; printf \"Namespace ${BOOKBUYER_NAMESPACE}:\n\"; kubectl get pods -n ${BOOKBUYER_NAMESPACE} -o wide; printf \"\n\n\"; printf \"Namespace ${BOOKSTORE_NAMESPACE}:\n\"; kubectl get pods -n ${BOOKSTORE_NAMESPACE} -o wide; printf \"\n\n\"; printf \"Namespace ${BOOKTHIEF_NAMESPACE}:\n\"; kubectl get pods -n ${BOOKTHIEF_NAMESPACE} -o wide; printf \"\n\n\"; printf \"Namespace ${BOOKWAREHOUSE_NAMESPACE}:\n\"; kubectl get pods -n ${BOOKWAREHOUSE_NAMESPACE} -o wide"
fi
