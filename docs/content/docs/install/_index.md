---
title: "Installation"
description: "This section describes how to install/uninstall Open Service Mesh (OSM) on a Kubernetes cluster using the `osm` CLI."
type: docs
aliases: ["installation guide"]
weight: 3
---

## Prerequisites

- Kubernetes cluster running Kubernetes v1.15.0 or greater

## Set up the OSM CLI

### From the Binary Releases

Download platform specific compressed package from the [Releases page](https://github.com/openservicemesh/osm/releases).
Unpack the `osm` binary and add it to `$PATH` to get started.

### From Source (Linux, MacOS)

Building OSM from source requires more steps but is the best way to test the latest changes and useful in a development environment.

You must have a working [Go](https://golang.org/doc/install) environment.

```console
$ git clone git@github.com:openservicemesh/osm.git
$ cd osm
$ make build-osm
```

`make build-osm` will fetch any required dependencies, compile `osm` and place it in `bin/osm`. Add `bin/osm` to `$PATH` so you can easily use `osm`.

## Install OSM

Use the `osm` CLI to install the OSM control plane on to a Kubernetes cluster.

Run `osm install`.

```console
# Install osm control plane components
$ osm install
OSM installed successfully in namespace [osm-system] with mesh name [osm]
```

By default, the control plane components are installed into a Kubernetes Namespace called `osm-system` and the control plane is given a unique identifier attribute `mesh-name` defaulted to `osm`. Both the Namespace and mesh-name can be configured with flags to the `osm install` command. Running `osm install --help` provides details on the various flags that can be configured.

The `mesh-name` is a unique identifier assigned to an osm-controller instance during install to identify and manage a mesh instance.

The `mesh-name` should follow [RFC 1123](https://tools.ietf.org/html/rfc1123) DNS Label constraints. The `mesh-name` must:

- contain at most 63 characters
- contain only lowercase alphanumeric characters or '-'
- start with an alphanumeric character
- end with an alphanumeric character

## Inspect OSM Components

A few components will be installed by default into the `osm-system` Namespace. Inspect them by using the following `kubectl` command:

```console
$ kubectl get pods,svc,secrets,configmaps,serviceaccount --namespace osm-system
```

A few cluster wide (non Namespaced components) will also be installed. Inspect them using the following `kubectl` command:

```console
kubectl get clusterrolebinding,clusterrole,mutatingwebhookconfiguration
```

Under the hood, `osm` is using [Helm](https://helm.sh) libraries to create a Helm `release` object in the control plane Namespace. The Helm `release` name is the mesh-name. The `helm` CLI can also be used to inspect Kubernetes manifests installed in more detail. Goto https://helm.sh for instructions to install Helm.

```console
$ helm get manifest osm --namespace osm-system
```

## Next Steps

Now that the OSM control plane is up and running, [add services](../onboard_services/) to the mesh.
