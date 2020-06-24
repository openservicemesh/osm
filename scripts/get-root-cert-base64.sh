#!/bin/bash

# shellcheck disable=SC1091
source .env

kubectl get secret osm-ca-bundle -n "$K8S_NAMESPACE" -o json | jq -r '.data["ca.crt"]' | base64 -d

# TODO
# az network application-gateway root-cert create --cert-file test.cer  --gateway-name $appgwName  --name name-of-my-root-cert1 --resource-group $resgp
