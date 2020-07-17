#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# desired cluster name; default is "kind"
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-osm}"

if kind get clusters | grep -q ^osm$ ; then
  echo "cluster already exists, moving on"
  exit 0
fi

# create registry container unless it already exists
kind_version=$(kind version)
kind_network='kind'
reg_name='kind-registry'
reg_port='5000'
case "${kind_version}" in
  "kind v0.7."* | "kind v0.6."* | "kind v0.5."*)
    kind_network='bridge'
    ;;
esac

# create registry container unless it already exists
running="$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)"
if [ "${running}" != 'true' ]; then
  docker run \
    -d --restart=always -p "${reg_port}:5000" --name "${reg_name}" \
    registry:2
fi

reg_host="${reg_name}"
if [ "${kind_network}" = "bridge" ]; then
    reg_host="$(docker inspect -f '{{.NetworkSettings.IPAddress}}' "${reg_name}")"
fi
echo "Registry Host: ${reg_host}"

# create a cluster with the local registry enabled in containerd
cat <<EOF | kind create cluster --name "${KIND_CLUSTER_NAME}" --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:${reg_port}"]
    endpoint = ["http://${reg_host}:${reg_port}"]
EOF

for node in $(kind get nodes --name "${KIND_CLUSTER_NAME}"); do
  kubectl annotate node "${node}" tilt.dev/registry=localhost:${reg_port};
done

if [ "${kind_network}" != "bridge" ]; then
  containers=$(docker network inspect ${kind_network} -f "{{range .Containers}}{{.Name}} {{end}}")
  needs_connect="true"
  for c in $containers; do
    if [ "$c" = "${reg_name}" ]; then
      needs_connect="false"
    fi
  done
  if [ "${needs_connect}" = "true" ]; then               
    docker network connect "${kind_network}" "${reg_name}" || true
  fi
fi
