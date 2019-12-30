#!/bin/bash

source .env

POD=$(kubectl get pods -n "$K8S_NAMESPACE" --show-labels --selector app=bookbuyer --no-headers | grep -v 'Terminating' | awk '{print $1}' | head -n1)

kubectl logs $POD -n "$K8S_NAMESPACE" -c bookbuyer --tail=100 -f
