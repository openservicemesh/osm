#!/bin/bash

POD="$(kubectl get pods --selector app=bookbuyer -nsmc --no-headers | grep 'Running' | awk '{print $1}')"

kubectl port-forward "$POD" -n osm 15000:15000

