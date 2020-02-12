#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

NAME="sds"
PORT=15123

./demo/deploy-xds.sh $NAME $PORT
