#!/bin/bash

set -aueo pipefail

envoy \
    --log-level debug \
    -c ./demo/config/localhost-eds.yaml \
    --service-node local-test \
    --service-cluster local-test
