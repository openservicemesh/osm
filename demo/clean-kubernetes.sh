#!/bin/bash

set -aueo pipefail

NAMESPACE="smc"

kubectl delete namespace "$NAMESPACE" || true
kubectl create namespace "$NAMESPACE" || true

kubectl get all -n "$NAMESPACE"

for x in $(kubectl get deployments -n "$NAMESPACE" --no-headers | awk '{print $1}');  do
    kubectl delete deployment $x -n "$NAMESPACE"
done

for x in $(kubectl get services -n "$NAMESPACE" --no-headers | awk '{print $1}');  do
    kubectl delete service $x -n "$NAMESPACE"
done

for x in $(kubectl get configmaps -n "$NAMESPACE" --no-headers | awk '{print $1}');  do
    kubectl delete configmap $x -n "$NAMESPACE"
done
