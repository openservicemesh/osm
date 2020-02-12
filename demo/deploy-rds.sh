#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

NAME="rds"
PORT=15126

./demo/deploy-xds.sh $NAME $PORT
