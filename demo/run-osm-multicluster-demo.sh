#!/bin/bash

# Automated Multicluster Demo
# This script will deploy Open Service Mesh on two appointed kubernetes clusters.
# The result will be, bookbuyer service on cluster can communicate with the bookstore service on another cluster.

set -aueo pipefail

if [ ! -f .env ]; then
    echo -e "\nThere is no .env file in the root of this repository."
    echo -e "Copy the values from .env.example into .env."
    echo -e "Modify the values in .env to match your setup.\n"
    echo -e "    cat .env.example > .env\n\n"
    exit 1
fi

TIMEOUT="600s"

# shellcheck disable=SC1091
source .env

# Set meaningful defaults for env vars we expect from .env
MESH_NAME="${MESH_NAME:-osm}"
K8S_NAMESPACE="${K8S_NAMESPACE:-osm-system}"
BOOKBUYER_NAMESPACE="${BOOKBUYER_NAMESPACE:-bookbuyer}"
BOOKSTORE_NAMESPACE="${BOOKSTORE_NAMESPACE:-bookstore}"
BOOKTHIEF_NAMESPACE="${BOOKTHIEF_NAMESPACE:-bookthief}"
BOOKWAREHOUSE_NAMESPACE="${BOOKWAREHOUSE_NAMESPACE:-bookwarehouse}"
CTR_REGISTRY="${CTR_REGISTRY:-localhost:5000}"
CTR_REGISTRY_CREDS_NAME="${CTR_REGISTRY_CREDS_NAME:-acr-creds}"
DEPLOY_TRAFFIC_SPLIT="${DEPLOY_TRAFFIC_SPLIT:-true}"
CTR_TAG="${CTR_TAG:-$(git rev-parse HEAD)}"
IMAGE_PULL_POLICY="${IMAGE_PULL_POLICY:-Always}"
ENABLE_DEBUG_SERVER="${ENABLE_DEBUG_SERVER:-true}"
ENABLE_EGRESS="${ENABLE_EGRESS:-false}"
ENABLE_RECONCILER="${ENABLE_RECONCILER:-false}"
DEPLOY_GRAFANA="${DEPLOY_GRAFANA:-false}"
DEPLOY_JAEGER="${DEPLOY_JAEGER:-false}"
ENABLE_FLUENTBIT="${ENABLE_FLUENTBIT:-false}"
DEPLOY_PROMETHEUS="${DEPLOY_PROMETHEUS:-false}"
DEPLOY_WITH_SAME_SA="${DEPLOY_WITH_SAME_SA:-false}"
ENVOY_LOG_LEVEL="${ENVOY_LOG_LEVEL:-debug}"
DEPLOY_ON_OPENSHIFT="${DEPLOY_ON_OPENSHIFT:-false}"
USE_PRIVATE_REGISTRY="${USE_PRIVATE_REGISTRY:-true}"
ALPHA_CLUSTER="${ALPHA_CLUSTER:-alpha}"
BETA_CLUSTER="${BETA_CLUSTER:-beta}"
MULTICLUSTER_CONTEXTS="${MULTICLUSTER_CONTEXTS:-$ALPHA_CLUSTER $BETA_CLUSTER}"

# For any additional installation arguments. Used heavily in CI.
optionalInstallArgs=$*

exit_error() {
    error="$1"
    echo "$error"
    exit 1
}

# Because this script is specifically for launching the OSM Multicluster demo
# we can set the ENABLE_MULTICLUSTER variabl to "true"
export ENABLE_MULTICLUSTER=true

# As of 2021-08-04 we need to set this for the bookbuyer to be able to resolve bookstore svc
# in multicluster setups
export BOOKSTORE_SVC=bookstore-v1


# Check if Docker daemon is running
docker info > /dev/null || { echo "Docker daemon is not running"; exit 1; }

# Build OSM binaries
make build-osm

# Push to registry - needs to happen after registry creation
make docker-build


echo "Kubernetes contexts to be deployed to: $MULTICLUSTER_CONTEXTS"

for CONTEXT in $MULTICLUSTER_CONTEXTS; do
    kubectl config use-context "$CONTEXT"

    # cleanup stale resources from previous runs
    ./demo/clean-kubernetes.sh

    # The demo uses osm's namespace as defined by environment variables, K8S_NAMESPACE
    # to house the control plane components.
    #
    # Note: `osm install` creates the namespace via Helm only if such a namespace already
    # doesn't exist. We explicitly create the namespace below because of the need to
    # create container registry credentials in this namespace for the purpose of testing.
    # The side effect of creating the namespace here instead of letting Helm create it is
    # that Helm no longer manages namespace creation, and as a result labels that it
    # otherwise adds for using as a namespace selector are no longer available.
    kubectl create namespace "$K8S_NAMESPACE"
    # Mimic Helm namespace label behavior: https://github.com/helm/helm/blob/release-3.2/pkg/action/install.go#L292
    kubectl label namespace "$K8S_NAMESPACE" name="$K8S_NAMESPACE"

    if [ "$DEPLOY_ON_OPENSHIFT" = true ] ; then
        optionalInstallArgs+=" --set=osm.enablePrivilegedInitContainer=true"
    fi

    # Registry credentials
    ./scripts/create-container-registry-creds.sh "$K8S_NAMESPACE"

    # Copy osm-ca-bundle from $ALPHA_CLUSTER to $BETA_CLUSTER
    # For multicluster : A common root certificate is needed for all clusters installing OSM
    # TODO: Certificate story for Multicluster TBD : Issue #3446
    if [ "${CONTEXT}" = "$BETA_CLUSTER" ]; then
      ./demo/copy-osm-ca-bundle.sh
    fi

    # shellcheck disable=SC2086
    bin/osm install \
        --osm-namespace "$K8S_NAMESPACE" \
        --verbose \
        --mesh-name "$MESH_NAME" \
        --set=osm.image.registry="$CTR_REGISTRY" \
        --set=osm.imagePullSecrets[0].name="$CTR_REGISTRY_CREDS_NAME" \
        --set=osm.image.tag="$CTR_TAG" \
        --set=osm.image.pullPolicy="$IMAGE_PULL_POLICY" \
        --set=osm.enableDebugServer="$ENABLE_DEBUG_SERVER" \
        --set=osm.enableEgress="$ENABLE_EGRESS" \
        --set=osm.enableReconciler="$ENABLE_RECONCILER" \
        --set=osm.deployGrafana="$DEPLOY_GRAFANA" \
        --set=osm.deployJaeger="$DEPLOY_JAEGER" \
        --set=osm.enableFluentbit="$ENABLE_FLUENTBIT" \
        --set=osm.deployPrometheus="$DEPLOY_PROMETHEUS" \
        --set=osm.envoyLogLevel="$ENVOY_LOG_LEVEL" \
        --set=osm.controllerLogLevel="trace" \
        --set=osm.featureFlags.enableMulticlusterMode="true" \
        --set=osm.featureFlags.enableEnvoyActiveHealthChecks="true" \
        --timeout="$TIMEOUT" \
        $optionalInstallArgs

    ./demo/configure-app-namespaces.sh

    if [ "${CONTEXT}" = "$ALPHA_CLUSTER" ]; then
        echo -e "Install the Bookbuyer artifacts in the ${CONTEXT} cluster"
        ./demo/deploy-bookbuyer.sh

        echo -e "Install a Bookstore-v1 in the ${CONTEXT} cluster"
        ./demo/deploy-bookstore.sh "v1"
    fi

    if [ "${CONTEXT}" = "$BETA_CLUSTER" ]; then
        echo -e "Install a Bookstore-v1 in the ${CONTEXT} cluster"
        ./demo/deploy-bookstore.sh "v1"
    fi
done

# create bookstore-v1 multicluster service object in $ALPHA_CLUSTER cluster only
# this will help bookbuyer identify the existance of bookstore-v1 in $BETA_CLUSTER
./demo/deploy-MultiClusterService.sh "v1"

for CONTEXT in $MULTICLUSTER_CONTEXTS; do
     # Apply SMI policies on all clusters
     kubectl config use-context "$CONTEXT"
    ./demo/deploy-multicluster-smi-policies.sh
done

echo "Switching to $ALPHA_CLUSTER"
kubectl config use-context "$ALPHA_CLUSTER"

echo "Injecting unavailable workloads to $BOOKSTORE_NAMESPACE/$BOOKBUYER_SVC"
./demo/multicluster-fault-injection.sh apply
echo "Run './demo/multicluster-fault-injection.sh delete' after the demo if you want to remove the injected faulty workloads."

echo "Bookbuyer logs... (showing identity of responding bookstore pods)"
sleep 2
kubectl logs -n bookbuyer --selector app=bookbuyer -c bookbuyer -f | grep Identity
