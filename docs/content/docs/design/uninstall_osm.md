---
title: "Uninstall OSM"
description: "Uninstall OSM Control Plane from Cluster"
type: docs
---

# Uninstall OSM Control Plane from Cluster

- **Revision:** 1
- **Status:** Implemented

## Table of Contents
<!-- toc -->
- [Summary](#summary)
  - [Use Case](#use-case)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Experience](#experience)
  - [Solution](#solution)
    - [Implementation Details](#implementation-details)
  - [Testing](#testing)
- [Historical Context](#historical-context)
- [Looking Forward](#looking-forward)
<!-- /toc -->

## Summary
A user should be able to delete all Kubernetes resources related to an OSM control plane installation using a CLI command.

### Use Case
A user can use `osm install` to install an OSM control plane in a Kubernetes cluster to test out OSM. Once tested, they may want to uninstall the control plane components.

## Motivation
Make it easier to cleanup OSM control planes in test clusters

### Goals
Delete all Kubernetes resources associated with OSM control plane.

### Non goals
- This version of the proposal will not handle removing any CRDs since these are cluster-wide resources that may be used by other OSM control planes, service mesh implementations or tools.
- This version of the proposal will not handle removing Envoy proxies from application pods. The user must re-start application Pods once the `osm` control plane has been uninstalled. Traffic will be interrupted.
- This version does not remove the `openservicemesh.io/monitored-by` label from the Namespaces observed by an OSM control plane.

## Proposal

### Experience
CLI command:
```bash
osm mesh uninstall MESH_NAME
```

### Solution
The CLI command will uninstall the `osm-controller` Deployment, Service and any other resources associated with the OSM control plane with the given MESH_NAME.

#### Implementation Details
The `osm` CLI uses Helm libraries and a Helm chart under the hood to install an OSM control plane in a Kubernetes cluster. It also creates a namespace (`osm-system` by default) to house the Kubernetes resources defined in and installed using the Helm chart. The `osm mesh uninstall` command works by deleting the Helm release associated with the control plane. The Helm release lives in the same namespace as the OSM control plane installation. The namespace will not be deleted because we don't know what else for now the user has installed in that namespace. Because the SMI CRDs live in the chart's `crds/` directory, they are protected from any delete action in Helm.

### Testing
This command and all associated functionality should be unit tested. This command should be added to the simulations run by CI.

## Historical Context
- [Issue #636](https://github.com/openservicemesh/osm/issues/636)
- [Issue #927](https://github.com/openservicemesh/osm/issues/927)
- [Issue #1023](https://github.com/openservicemesh/osm/issues/1023)

## Looking Forward
In future iterations of this feature, we should:
- have an option to delete the CRDs associated with OSM
- have an option to delete the namespace associate with the OSM control plane
- remove namespace labels/annotations associated with the OSM control plane
- have an automated way of removing the Envoy proxy and any OSM specific labels/annotations from application Pods
