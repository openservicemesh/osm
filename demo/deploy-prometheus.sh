#!/bin/bash

set -auexo pipefail

./demo/metrics/prometheus/deploy-clusterRole.sh
./demo/metrics/prometheus/deploy-configMap.sh
./demo/metrics/prometheus/deploy-prometheusService.sh