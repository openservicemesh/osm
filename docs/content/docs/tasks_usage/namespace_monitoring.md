---
title: "Namespace Monitoring"
description: "This section describes how and why OSM monitors Kubernetes namespaces"
type: docs
---

## Overview

When setting up an OSM control plane (also referred to as a "mesh"), one can also enroll a set of Kubernetes namespaces to the mesh. Enrolling a namespace to OSM allows OSM to monitor the resources within that Namespace whether they be applications deployed in Pods, Services, or even traffic policies represented as SMI resources.

Only one mesh can monitor a namespace, so this is something to watch out for when there are multiple instances of OSM within the same Kubernetes cluster. When applying policies to applications, OSM will only assess resources in either monitored namespaces so it is important to enroll namespaces where your applications are deployed to the correct instance of OSM with the correct mesh name.
Enrolling a namespace also optionally allows for metrics to be collected for resources in the given namespace and for Pods in the namespace to be automatically injected with sidecar proxy containers. These are all features that help OSM provide functionality for traffic management and observability. Scoping this functionality at the namespace level allows teams to organize which segments of
their cluster should be part of which mesh.

Namespace monitoring, automatic sidecar injection, and metrics collection is controlled by adding certain labels and annotations to a Kubernetes namespace. This can be done manually or using the `osm` CLI although using the `osm` CLI is the recommended approach. The presence of the label `openservicemesh.io/monitored-by=<mesh-name>` allows an OSM control place with the given `mesh-name` to monitor
all resources within that namespace. The annotation `openservicemesh.io/sidecar-injection=enabled` enables OSM to automatically inject sidecar proxy containers in all Pods created within that namespace. The metrics annotation `openservicemesh.io/metrics=enabled` allows OSM to collect metrics on resources within a Namespace.

See how to use the OSM CLI to manage namespace monitoring below.

## Adding a Namespace to the OSM Control Plane

Add a namespace for monitoring and sidecar injection to the mesh with the following command:

```bash
osm namespace add <namespace>
```

Explicitly disable sidecar injection while adding the namespace using `--disable-sidecar-injection` flag as shown [here](../tasks_usage/sidecar_injection.md#Explicitly-Disabling-Automatic-Sidecar-Injection-on-Namespaces).

## Remove a Namespace from the OSM control plane

Remove a namespace from being monitored by the mesh and disable sidecar injection with the following command:

```bash
osm namespace remove <namespace>
```

This command will remove the OSM specific labels and annotations on the namespace thus removing it from the mesh.

## Enable Metrics for a Namespace

```bash
osm metrics enable --namespace <namespace>
```

## Ignore a Namespace

There may be namespaces in a cluster that should never be part of a mesh. To explicity exclude a namespace from OSM:

``` bash
osm namespace ignore <namespace>
```

## List Namespaces Part of a Mesh

To list namespaces within a specific mesh:

```bash
osm namespace list --mesh-name=<mesh-name>
```

## Troubleshooting Guide

### Policy Issues

If you're not seeing changes in SMI policies being applied to resources in a namespace, ensure the namespace is enrolled in the correct mesh:

```bash
osm namespace list --mesh-name=<mesh-name>

NAMESPACE         MESH   SIDECAR-INJECTION
<namespace>       osm    enabled
```

If the namespace does not show up, check the labels on the namespace using `kubectl`:

```bash
kubectl get namespace <namespace> --show-labels

NAME          STATUS   AGE   LABELS
<namespace>   Active   36s   openservicemesh.io/monitored-by=<mesh-name>
```

If the label value is not the expected `mesh-name`, remove the namespace from the mesh and add it back using the correct `mesh-name`.

```bash
osm namespace remove <namespace> --mesh-name=<current-mesh-name>
osm namespace add <namespace> --mesh-name=<expected-mesh-name>
```

If the monitored-by label is not present, it was either not added to the mesh or there was an error when adding it to the mesh.
Add the namespace to the mesh either with the `osm` CLI or using kubectl:

```bash
osm namespace add <namespace> --mesh-name=<mesh-name>
```

```bash
kubectl label namespace <namespace> openservicemesh.io/monitored-by=<mesh-name>
```

### Issues with Automatic Sidecar Injection

If you're not seeing your Pods being automatically injected with sidecar containers, ensure that sidecar injection is enabled:

```bash
osm namespace list --mesh-name=<mesh-name>

NAMESPACE         MESH   SIDECAR-INJECTION
<namespace>       osm    enabled
```

If the namespace does not show up, check the annotations on the namespace using `kubectl`:

```bash
kubectl get namespace <namespace> -o=jsonpath='{.metadata.annotations.openservicemesh\.io\/sidecar-injection}'
```

If the output is anything other than `enabled`, either add namespace using the `osm` CLI or add the annotation with `kubectl`:

```bash
osm namespace add <namespace> --mesh-name=<mesh-name> --disable-sidecar-injection=false
```

```bash
kubectl annotate namespace <namespace> openservicemesh.io/sidecar-injection=enabled --overwrite
```

### Issues with Metrics Collection

If you're not seeing metrics for resources in a particular namespace, ensure metrics are enabled:

```bash
kubectl get namespace <namespace> -o=jsonpath='{.metadata.annotations.openservicemesh\.io\/metrics}'
```

If the output is anything other than `enabled`, enable the namespace usng the `osm` CLI or add the annotation with `kubectl`:

```bash
osm metrics enable --namespace <namespace>
```

```bash
kubectl annotate namespace <namespace> openservicemesh.io/metrics=enabled --overwrite
```

### Other Issues

If you're running into issues that have not been resolved with the debugging techniques above, please open a GitHub issue on the repository.
