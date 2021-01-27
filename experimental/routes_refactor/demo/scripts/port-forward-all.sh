#!/bin/bash

./scripts/port-forward-bookbuyer-ui.sh &
./scripts/port-forward-bookstore-ui-v1.sh &
./scripts/port-forward-bookstore-ui-v2.sh &
./scripts/port-forward-bookthief-ui.sh &

./scripts/port-forward-bookbuyer-envoy.sh &
./scripts/port-forward-bookstore-v1-envoy.sh &
./scripts/port-forward-bookstore-v2-envoy.sh &
./scripts/port-forward-bookthief-envoy.sh &

wait