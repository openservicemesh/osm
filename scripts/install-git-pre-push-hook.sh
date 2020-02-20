#!/bin/bash

mkdir -p .git/hooks
pushd .git/hooks && ln -f -s ../../scripts/pre-push-hook ./pre-push && popd
chmod +x .git/hooks/pre-push
