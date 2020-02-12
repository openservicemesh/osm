#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

NAME="eds"
PORT=15124

./demo/deploy-xds.sh $NAME $PORT
