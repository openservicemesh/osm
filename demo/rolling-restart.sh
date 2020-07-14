#!/bin/bash

# This script performs a rolling restart of the deployments listed below.
# This is part of the OSM Bookstore demo helper scripts.

kubectl rollout restart deployment -n bookbuyer       bookbuyer
kubectl rollout restart deployment -n bookstore       bookstore-v1
kubectl rollout restart deployment -n bookstore       bookstore-v2
kubectl rollout restart deployment -n bookthief       bookthief
kubectl rollout restart deployment -n bookwarehouse   bookwarehouse
