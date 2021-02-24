---
title: "Sidecar Injection"
description: "This section describes the sidecar injection workflow in OSM."
type: docs
---

# Sidecar Injection
Services participating in the service mesh communicate via sidecar proxies installed on pods backing the services. The following sections describe the sidecar injection workflow in OSM.

## Automatic Sidecar Injection
Automatic sidecar injection is currently the only way to inject sidecars into the service mesh. Sidecars can be automatically injected into applicable Kubernetes pods using a mutating webhook admission controller provided by OSM.

Automatic sidecar injection can be configured per namespace as a part of enrolling a namespace into the mesh, or later using the Kubernetes API. Automatic sidecar injection can be enabled either on a per namespace or per pod basis by annotating the namespace or pod resource with the sidecar injection annotation. Individual pods and namespaces can be explicitly configured to either enable or disable automatic sidecar injection, giving users the flexibility to control sidecar injection on pods and namespaces.

### Enabling Automatic Sidecar Injection

Prerequisites:
- The namespace to which the pods belong must be a monitored namespace that is added to the mesh using the `osm namespace add` command.
- The namespace to which the pods belong must not be set to be ignored using the `osm namespace ignore` command.
- The namespace to which the pods belong must not have a label with key `name` and value corresponding to the OSM control plane namespace. For example, a namespace with a label `name: osm-system` where `osm-system` is the control plane namespace cannot have sidecar injection enabled for pods in this namespace.

Automatic Sidecar injection can be enabled in the following ways:

- While enrolling a namespace into the mesh using `osm` cli: `osm namespace add <namespace>`:
  Automatic sidecar injection is enabled by default with this command.

- Using `kubectl` to annotate individual namespaces and pods to enable sidecar injection:

  ```console
  # Enable sidecar injection on a namespace
  $ kubectl annotate namespace <namespace> openservicemesh.io/sidecar-injection=enabled
  ```

  ```console
  # Enable sidecar injection on a pod
  $ kubectl annotate pod <pod> openservicemesh.io/sidecar-injection=enabled
  ```

- Setting the sidecar injection annotation to `enabled` in the Kubernetes resource spec for a namespace or pod:
  ```yaml
  metadata:
    name: test
    annotations:
      'openservicemesh.io/sidecar-injection': 'enabled'
  ```

  Pods will be injected with a sidecar only if the following conditions are met:
  1. The namespace to which the pod belongs is a monitored namespace.
  2. The pod is explicitly enabled for the sidecar injection, OR the namespace to which the pod belongs is enabled for the sidecar injection and the pod is not explicitly disabled for sidecar injection.

### Explicitly Disabling Automatic Sidecar Injection on Namespaces

Namespaces can be disabled for automatic sidecar injection in the following ways:

- While enrolling a namespace into the mesh using `osm` cli: `osm namespace add <namespace> --disable-sidecar-injection`:
  If the namespace was previously enabled for sidecar injection, it will be disabled after running this command.

- Using `kubectl` to annotate individual namespaces to disable sidecar injection:

  ```console
  # Disable sidecar injection on a namespace
  $ kubectl annotate namespace <namespace> openservicemesh.io/sidecar-injection=disabled
  ```

### Explicitly Disabling Automatic Sidecar Injection on Pods

Individual pods can be explicitly disabled for sidecar injection. This is useful when a namespace is enabled for sidecar injection but specific pods should not be injected with sidecars.

- Using `kubectl` to annotate individual pods to disable sidecar injection:
  ```console
  # Disable sidecar injection on a pod
  $ kubectl annotate pod <pod> openservicemesh.io/sidecar-injection=disabled
  ```

- Setting the sidecar injection annotation to `disabled` in the Kubernetes resource spec for the pod:
  ```yaml
  metadata:
    name: test
    annotations:
      'openservicemesh.io/sidecar-injection': 'disabled'
  ```

Automatic sidecar injection is implicitly disabled for a namespace when it is removed from the mesh using the `osm namespace remove` command.
