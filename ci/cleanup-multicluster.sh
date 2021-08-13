#!/bin/bash

for context in $MULTICLUSTER_CONTEXTS; do
    for ns in "$BOOKWAREHOUSE_NAMESPACE" "$BOOKBUYER_NAMESPACE" "$BOOKSTORE_NAMESPACE" "$BOOKTHIEF_NAMESPACE" "$K8S_NAMESPACE"; do
        kubectl delete namespace "$ns" --ignore-not-found --force --context "$context" &
    done
done