#!/bin/bash

set -aueo pipefail

source .env

POD=$(kubectl get pods -n "$K8S_NAMESPACE" --selector app=rds --no-headers | awk '{print $1}' | head -n1)

kubectl logs $POD -n "$K8S_NAMESPACE" -f
