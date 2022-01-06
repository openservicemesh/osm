#!/bin/bash

# This script is used to generate dummy embedded files for CI purposes.
if [ ! -f "cmd/cli/chart.tgz" ]; then
touch cmd/cli/chart.tgz
fi
