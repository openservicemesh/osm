#!/bin/bash

./experimental/routes_refactor/demo/scripts/port-forward-bookbuyer-ui.sh &
./experimental/routes_refactor/demo/scripts/port-forward-bookstore-ui-v1.sh &
./experimental/routes_refactor/demo/scripts/port-forward-bookstore-ui-v2.sh &
./experimental/routes_refactor/demo/scripts/port-forward-bookthief-ui.sh &

./experimental/routes_refactor/demo/scripts/port-forward-bookbuyer-envoy.sh &
./experimental/routes_refactor/demo/scripts/port-forward-bookstore-v1-envoy.sh &
./experimental/routes_refactor/demo/scripts/port-forward-bookstore-v2-envoy.sh &
./experimental/routes_refactor/demo/scripts/port-forward-bookthief-envoy.sh &

wait