#!/bin/bash

ORG_PATH="github.com/openservicemesh"
PROJECT_NAME="osm"
REPO_PATH="${ORG_PATH}/${PROJECT_NAME}"

VERSION_VAR="${REPO_PATH}/pkg/version.Version"
VERSION=$(git describe --abbrev=0 --tags)

DATE_VAR="${REPO_PATH}/pkg/version.BuildDate"
BUILD_DATE=$(date +%Y-%m-%d-%H:%MT%z)

COMMIT_VAR="${REPO_PATH}/pkg/version.GitCommit"
GIT_HASH=$(git rev-parse --short HEAD)

GOOS=linux GOBIN="$(pwd)/bin" go install -ldflags "-s -X ${VERSION_VAR}=${VERSION} -X ${DATE_VAR}=${BUILD_DATE} -X ${COMMIT_VAR}=${GIT_HASH}" -v ./cmd/osm-controller

