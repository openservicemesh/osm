#!/bin/bash
# shellcheck disable=SC1091

IMAGE_NAME="$1"
OS="$2"
IMAGE_REPO="$3"

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

tokenUri="https://auth.docker.io/token?service=registry.docker.io&scope=repository:$IMAGE_REPO/$IMAGE_NAME:pull"
bearerToken="$(curl --silent --get "$tokenUri" | jq --raw-output '.token')"
listUri="https://registry-1.docker.io/v2/$IMAGE_REPO/$IMAGE_NAME/tags/list"
authz="Authorization: Bearer $bearerToken"
version_list="$(curl --silent --get -H "Accept: application/json" -H "$authz" "$listUri" | jq --raw-output '.')"
exists=$(echo "$version_list" | jq --arg t "${CTR_TAG}" '.tags | index($t)')

if [[ $exists == null ]]
then
    if [[ $OS == "linux" ]]; then
        make docker-build-"$IMAGE_NAME"
        docker push "$IMAGE_REPO/$IMAGE_NAME:${CTR_TAG}" || { echo "Error pushing images to container registry $CTR_REGISTRY/$IMAGE_NAME:$CTR_TAG"; exit 1; }
    else
        make ARGS=--push "docker-build-$IMAGE_NAME"
    fi
fi
