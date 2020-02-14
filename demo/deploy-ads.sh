#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

NAME="ads"
PORT=15128

./demo/deploy-xds.sh $NAME $PORT
