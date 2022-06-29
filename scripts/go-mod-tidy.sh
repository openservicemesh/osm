#!/bin/bash

go mod tidy
if ! git diff --exit-code go.mod; then
    echo -e "\nPlease commit the changes made by 'go mod tidy'"
    exit 1
fi

if ! git diff --exit-code go.mod; then
    echo -e "\nPlease commit the changes made by 'go mod tidy'"
    exit 1
fi
