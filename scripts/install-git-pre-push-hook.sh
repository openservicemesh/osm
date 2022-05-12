#!/bin/bash

mkdir -p .git/hooks

# shellcheck disable=SC2164
pushd .git/hooks && ln -f -s ../../scripts/pre-push-hook ./pre-push && popd

chmod +x .git/hooks/pre-push
