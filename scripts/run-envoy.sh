#!/bin/bash

set -aueo pipefail

envoy \
    --log-level debug \
    -c ./demo/config/localhost-eds.yaml
