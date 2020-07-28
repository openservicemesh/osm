#!/bin/bash

POD=$(kubectl get pods -n calculator --selector app=calculator-front-end --no-headers | awk '{print $1}')

kubectl port-forward "$POD" -n calculator 8080:8080
