---
title: "Installation"
description: "This section describes how to install/uninstall Open Service Mesh (OSM) on a Kubernetes cluster using the `osm` CLI."
type: docs
aliases: ["installation guide"]
weight: 3
---

## Prerequisites

- Kubernetes cluster running Kubernetes v1.18.0 or greater
- The [osm CLI](#set-up-the-osm-cli) or the [helm 3 CLI](https://helm.sh/docs/intro/install/)

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

### OSM Configuration
By default, the control plane components are installed into a Kubernetes Namespace called `osm-system` and the control plane is given a unique identifier attribute `mesh-name` defaulted to `osm`. Both the Namespace and mesh-name can be configured through flags when using the `osm CLI` flags or by editing the values file when using the `helm CLI`.

The `mesh-name` is a unique identifier assigned to an osm-controller instance during install to identify and manage a mesh instance.

The `mesh-name` should follow [RFC 1123](https://tools.ietf.org/html/rfc1123) DNS Label constraints. The `mesh-name` must:
- contain at most 63 characters
- contain only lowercase alphanumeric characters or '-'
- start with an alphanumeric character
- end with an alphanumeric character

### Using the OSM CLI
Use the `osm` CLI to install the OSM control plane on to a Kubernetes cluster.

Run `osm install`.

```console
# Install osm control plane components
$ osm install
OSM installed successfully in namespace [osm-system] with mesh name [osm]
```

Run `osm install --help` for more options.

### Using the Helm CLI
The [OSM chart](https://github.com/openservicemesh/osm/tree/release-v0.8/charts/osm) can be installed directly via the [Helm CLI](https://helm.sh/docs/intro/install/).

#### Editing the Values File
You can configure the OSM installation by overriding the values file.
1. Create a copy of the [values file](https://github.com/openservicemesh/osm/blob/release-v0.8/charts/osm/values.yaml) (make sure to use the version for the chart you wish to install).
1. Change any values you wish to customize. You can omit all other values.
   - To see which values correspond to the ConfigMap settings, see the [OSM ConfigMap documentation](../osm_config_map.md)

   - For example, to set the `envoy_log_level` field in the ConfigMap to `info`, save the following as `override.yaml`:
     ```
     OpenServiceMesh:
       envoyLogLevel: info
     ```

#### Helm install
Then run the following `helm install` command. The chart version can be found in the Helm chart you wish to install [here](https://github.com/openservicemesh/osm/blob/release-v0.8/charts/osm/Chart.yaml#L17).

```console
$ helm install <mesh name> osm --repo https://openservicemesh.github.io/osm --version <chart version> --namespace <osm namespace> --values override.yaml
```

Omit the `--values` flag if you prefer to use the default settings.

Run `helm install --help` for more options.

### OpenShift

To install OSM on OpenShift:
1. Enable privileged init containers so that they can properly program iptables. The NET_ADMIN capability is not sufficient on OpenShift.
    ```shell
    osm install --set="OpenServiceMesh.enablePrivilegedInitContainer=true"
    ```
    - If you have already installed OSM without enabling privileged init containers, set `enable_privileged_init_container` to `true` in the [OSM ConfigMap](../osm_config_map.md) and restart any pods in the mesh.
1. Add the `privileged` [security context constraint](https://docs.openshift.com/container-platform/4.7/authentication/managing-security-context-constraints.html) to each service account in the mesh.
    - Install the [oc CLI](https://docs.openshift.com/container-platform/4.7/cli_reference/openshift_cli/getting-started-cli.html).
    - Add the security context constraint to the service account
       ```shell
        oc adm policy add-scc-to-user privileged -z <service account name> -n <service account namespace>
       ```

### Pod Security Policy

OSM support for Pod Security Policy is still a work in progress. Some features may not be fully supported. Any issues can be filed in the [OSM GitHub repo](https://github.com/openservicemesh/osm/issues).

If you are running OSM in a cluster with PSPs enabled, pass in `--set OpenServiceMesh.pspEnabled=true` to your `osm install` or `helm install` CLI command.

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

Now that the OSM control plane is up and running, [add services](../tasks_usage/onboard_services.md) to the mesh.
