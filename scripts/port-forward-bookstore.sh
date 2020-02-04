#!/bin/bash

kubectl port-forward $(kubectl get pods --selector app=bookstore-1 -nsmc --no-headers | grep 'Running' | awk '{print $1}') -n smc 15000:15000

