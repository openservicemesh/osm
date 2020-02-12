#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

NAME="lds"
PORT=15127

./demo/deploy-xds.sh $NAME $PORT
