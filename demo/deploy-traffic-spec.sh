#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

kubectl apply -f - <<EOF
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: httproutegroups.specs.smi-spec.io
spec:
  group: specs.smi-spec.io
  version: v1alpha1
  scope: Namespaced
  names:
    kind: HTTPRouteGroup
    shortNames:
      - htr
    plural: httproutegroups
    singular: httproutegroup

---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: tcproutes.specs.smi-spec.io
spec:
  group: specs.smi-spec.io
  version: v1alpha1
  scope: Namespaced
  names:
    kind: TCPRoute
    shortNames:
      - tr
    plural: tcproutes
    singular: tcproute

---

apiVersion: specs.smi-spec.io/v1alpha1
kind: HTTPRouteGroup
metadata:
  name: bookstore-service-routes
  namespace: "$K8S_NAMESPACE"
matches:
- name: counter
  pathRegex: /counter
  methods: ["GET"]
- name: incrementcounter
  pathRegex: /incrementcounter
  methods: ["GET"]
- name: updatecounter
  pathRegex: /updatecounter
  methods: ["POST"]
EOF
