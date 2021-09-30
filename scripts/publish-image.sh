#!/bin/bash
# shellcheck disable=SC1091

set -euo pipefail

IMAGE_NAME="$1"
OS="$2"
IMAGE_REPO="$3"
CTR_TAG="$4"
VERIFY_TAGS="${VERIFY_TAGS:-false}"

if [ -z "${IMAGE_NAME}" ]; then
    echo "Error: IMAGE_NAME not specified"
    exit 1
fi
if [ -z "${OS}" ]; then
    echo "Error: OS not specified"
    exit 1
fi
if [ -z "${IMAGE_REPO}" ]; then
    echo "Error: IMAGE_REPO not specified"
    exit 1
fi
if [ -z "${CTR_TAG}" ]; then
    echo "Error: CTR_TAG not specified"
    exit 1
fi

if [[ "$VERIFY_TAGS" == "true" ]]; then
    image="$IMAGE_NAME"
    if [[ $OS == "windows" ]]; then
      image="$image-windows"
    fi
    tokenUri="https://auth.docker.io/token?service=registry.docker.io&scope=repository:$IMAGE_REPO/$image:pull"
    bearerToken="$(curl --silent --get "$tokenUri" | jq --raw-output '.token')"
    listUri="https://registry-1.docker.io/v2/$IMAGE_REPO/$image/tags/list"
    authz="Authorization: Bearer $bearerToken"
    version_list="$(curl --silent --get -H "Accept: application/json" -H "$authz" "$listUri" | jq --raw-output '.')"
    exists=$(echo "$version_list" | jq --arg t "${CTR_TAG}" '.tags | index($t)')
    if [[ $exists != null ]]; then
        echo "image $IMAGE_REPO/$image:$CTR_TAG already exists and \$VERIFY_TAGS is set"
        exit 1
    fi
fi

if [[ $OS == "linux" ]]; then
    make "docker-build-$IMAGE_NAME"
    docker push "$IMAGE_REPO/$IMAGE_NAME:$CTR_TAG"
else
    make ARGS=--push "docker-build-windows-$IMAGE_NAME"
fi
