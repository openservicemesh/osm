#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

NAME="cds"
PORT=15125

./demo/deploy-xds.sh $NAME $PORT
