#!/bin/bash

go mod tidy
if ! git diff --exit-code go.mod go.sum ; then
    echo -e "\nPlease run 'go mod tidy' to clean up dependencies"
    exit 1
fi