#!/bin/bash

read -p "Enter BOOKBUYER pod local port:" -i "8080" -e BOOKBUYER_PORT
read -p "Enter BOOKSTOREv1 pod local port:" -i "8081" -e BOOKSTOREv1_PORT
read -p "Enter BOOKSTOREv2 pod local port:" -i "8082" -e BOOKSTOREv2_PORT
read -p "Enter BOOKTHIEF pod local port:" -i "8083" -e BOOKTHIEF_PORT

./scripts/port-forward-zipkin.sh &
./scripts/port-forward-bookbuyer-ui.sh $BOOKBUYER_PORT &
./scripts/port-forward-bookstore-ui-v2.sh bookstore-v2 $BOOKSTOREv2_PORT &
./scripts/port-forward-bookstore-ui-v1.sh bookstore-v1 $BOOKSTOREv1_PORT &
./scripts/port-forward-bookthief-ui.sh $BOOKTHIEF_PORT &
./scripts/port-forward-osm-debug.sh &
./scripts/port-forward-grafana.sh &

wait

