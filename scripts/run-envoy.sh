#!/bin/bash

envoy \
    --log-level debug \
    -c ./demo/config/localhost-eds.yaml
