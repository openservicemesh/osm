#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

WEBHOOK_NAME="osm-webhook-$1"

kubectl delete mutatingwebhookconfiguration "$WEBHOOK_NAME" --ignore-not-found=true
