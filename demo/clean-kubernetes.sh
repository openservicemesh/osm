#!/bin/bash

set -aueo pipefail

source .env

kubectl delete namespace "$K8S_NAMESPACE" || true
kubectl create namespace "$K8S_NAMESPACE" || true

kubectl get all -n "$K8S_NAMESPACE"

for x in $(kubectl get deployments -n "$K8S_NAMESPACE" --no-headers | awk '{print $1}');  do
    kubectl delete deployment $x -n "$K8S_NAMESPACE"
done

for x in $(kubectl get services -n "$K8S_NAMESPACE" --no-headers | awk '{print $1}');  do
    kubectl delete service $x -n "$K8S_NAMESPACE"
done

for x in $(kubectl get configmaps -n "$K8S_NAMESPACE" --no-headers | awk '{print $1}');  do
    kubectl delete configmap $x -n "$K8S_NAMESPACE"
done
