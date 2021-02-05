---
title: "Uninstall"
description: "Uninstall"
type: docs
aliases: ["uninstall"]
weight: 3
---

# Uninstallation Guide

This guide describes how to uninstall Open Service Mesh (OSM) from a Kubernetes cluster using the `osm` CLI.

## Prerequisites

- Kubernetes cluster with OSM installed
- The [osm CLI](../installation_guide/#set-up-the-osm-cli)

## Uninstall OSM

Use the `osm` CLI to uninstall the OSM control plane from a Kubernetes cluster.

Run `osm mesh uninstall`.

```console
# Uninstall osm control plane components
$ osm mesh uninstall
Uninstall OSM [mesh name: osm] ? [y/n]: y
OSM [mesh name: osm] uninstalled
```

Run `osm mesh uninstall --help` for more options.

## Resource Management

The following sections detail which Kubernetes resources are cleaned up and which remain after uninstalling OSM.

### Removed during OSM uninstallation

1. OSM controller resources (deployment, service, config map, and RBAC)
1. Prometheus, Grafana, Jaeger, and Fluentbit resources installed by OSM
1. Mutating webhook and validating webhook

### Remaining after OSM uninstallation

1. Existing Envoy sidecar containers
   - Redeploy application pods to delete sidecars
1. Envoy bootstrap config secrets (stored in the application namespace)
1. Namespace annotations, including but not limited to `openservicemesh.io/monitored-by`
1. Custom resource definitions ([CRDs](https://github.com/openservicemesh/osm/tree/main/charts/osm/crds))
1. Vault resources provided by the user
