#!/bin/bash
# shellcheck disable=SC1091

IMAGE_TAG=$1
IMAGE_REPO=openservicemesh

if [ -z "${CTR_TAG}" ]
then
    echo "Error CTR_TAG is empty"
    exit 1
fi

tokenUri="https://auth.docker.io/token?service=registry.docker.io&scope=repository:$IMAGE_REPO/$IMAGE_TAG:pull"
bearerToken="$(curl --silent --get "$tokenUri" | jq --raw-output '.token')"
listUri="https://registry-1.docker.io/v2/$IMAGE_REPO/$IMAGE_TAG/tags/list"
authz="Authorization: Bearer $bearerToken"
version_list="$(curl --silent --get -H "Accept: application/json" -H "$authz" "$listUri" | jq --raw-output '.')"
exists=$(echo "$version_list" | jq --arg t "${CTR_TAG}" '.tags | index($t)')

if [[ $exists == null ]]
then
    make docker-build-"$IMAGE_TAG"
    docker push "$IMAGE_REPO/$IMAGE_TAG:${CTR_TAG}" || { echo "Error pushing images to container registry $(CTR_REGISTRY)/$(NAME):$(CTR_TAG)"; exit 1; }
fi
