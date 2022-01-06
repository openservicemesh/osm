#!/bin/bash

set -euo pipefail

if [ -z "$1" ]; then
  echo "Error: expected one argument IMAGE:TAG"
  exit 1
fi

IMAGE=$(cut -d: -f1 <<< "$1")
TAG=$(cut -d: -f2 <<< "$1")

tokenUri="https://auth.docker.io/token?service=registry.docker.io&scope=repository:$IMAGE:pull"
bearerToken="$(curl --silent --get "$tokenUri" | jq --raw-output '.token')"
listUri="https://registry-1.docker.io/v2/$IMAGE/tags/list"
authz="Authorization: Bearer $bearerToken"
version_list="$(curl --silent --get -H "Accept: application/json" -H "$authz" "$listUri" | jq --raw-output '.')"
exists=$(echo "$version_list" | jq --arg t "${TAG}" '.tags | index($t)')
if [[ $exists != null ]]; then
  echo "image $IMAGE:$TAG already exists"
  exit 1
fi
