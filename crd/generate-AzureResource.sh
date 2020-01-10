#!/bin/bash

$GOPATH/src/k8s.io/code-generator/generate-groups.sh \
    all \
    github.com/deislabs/smc/pkg/smc_client \
    github.com/deislabs/smc/pkg/apis \
    "azureresource:v1"
