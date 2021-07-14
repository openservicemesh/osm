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
DEPLOY_GRAFANA="${DEPLOY_GRAFANA:-false}"
DEPLOY_JAEGER="${DEPLOY_JAEGER:-false}"
ENABLE_FLUENTBIT="${ENABLE_FLUENTBIT:-false}"
DEPLOY_PROMETHEUS="${DEPLOY_PROMETHEUS:-false}"
DEPLOY_WITH_SAME_SA="${DEPLOY_WITH_SAME_SA:-false}"
ENVOY_LOG_LEVEL="${ENVOY_LOG_LEVEL:-debug}"
DEPLOY_ON_OPENSHIFT="${DEPLOY_ON_OPENSHIFT:-false}"
MULTICLUSTER_CONTEXTS="${MULTICLUSTER_CONTEXTS:-()}"

# For any additional installation arguments. Used heavily in CI.
optionalInstallArgs=$*

exit_error() {
    error="$1"
    echo "$error"
    exit 1
}

# Check if Docker daemon is running
docker info > /dev/null || { echo "Docker daemon is not running"; exit 1; }

# Build OSM binaries
make build-osm

echo "Kubernetes contexts to be deployed to: $MULTICLUSTER_CONTEXTS"

for CONTEXT in $MULTICLUSTER_CONTEXTS; do
    # shellcheck disable=SC2034
    BOOKSTORE_CLUSTER_ID="$CONTEXT" # this is used when deploying bookstore app

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
        optionalInstallArgs+=" --set=OpenServiceMesh.enablePrivilegedInitContainer=true"
    fi

    # Push to registry - needs to happen after registry creation
    make docker-push

    # Registry credentials
    ./scripts/create-container-registry-creds.sh "$K8S_NAMESPACE"

    # shellcheck disable=SC2086
    bin/osm install \
        --osm-namespace "$K8S_NAMESPACE" \
        --mesh-name "$MESH_NAME" \
        --set=OpenServiceMesh.image.registry="$CTR_REGISTRY" \
        --set=OpenServiceMesh.imagePullSecrets[0].name="$CTR_REGISTRY_CREDS_NAME" \
        --set=OpenServiceMesh.image.tag="$CTR_TAG" \
        --set=OpenServiceMesh.image.pullPolicy="$IMAGE_PULL_POLICY" \
        --set=OpenServiceMesh.enableDebugServer="$ENABLE_DEBUG_SERVER" \
        --set=OpenServiceMesh.enableEgress="$ENABLE_EGRESS" \
        --set=OpenServiceMesh.deployGrafana="$DEPLOY_GRAFANA" \
        --set=OpenServiceMesh.deployJaeger="$DEPLOY_JAEGER" \
        --set=OpenServiceMesh.enableFluentbit="$ENABLE_FLUENTBIT" \
        --set=OpenServiceMesh.deployPrometheus="$DEPLOY_PROMETHEUS" \
        --set=OpenServiceMesh.envoyLogLevel="$ENVOY_LOG_LEVEL" \
        --set=OpenServiceMesh.controllerLogLevel="trace" \
        --set=OpenServiceMesh.featureFlags.enableMulticlusterMode="true" \
        --timeout="$TIMEOUT" \
        $optionalInstallArgs

    ./demo/configure-app-namespaces.sh

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
done
