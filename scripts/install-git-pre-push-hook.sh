#!/bin/bash

mkdir -p .git/hooks
pushd .git/hooks && ln -f -s ../../pre-push-hook ./pre-push && popd
chmod +x .git/hooks/pre-push
