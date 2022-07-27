#!/bin/bash

go mod tidy
if ! git diff --exit-code go.mod go.sum ; then
    echo -e "\nPlease commit the changes made by 'go mod tidy'"
    exit 1
fi
