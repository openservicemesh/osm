# Installation Guide

This guide describes how to install Open Service Mesh (OSM) on a Kubernetes cluster using the `osm` CLI.

## Prerequisites
- Kubernetes cluster running Kubernetes v1.15.0 or greater
- A private container registry (temporary requirement as this is currently a private repo)

## Set up the OSM CLI

### From the Binary Releases
Download platform specific compressed package from the [Releases page](https://github.com/open-service-mesh/osm/releases).
Unpack the `osm` binary and add it to `$PATH` to get started.

### From Source (Linux, MacOS)
Building OSM from source requires more steps but is the best way to test the latest changes and useful in a development environment.

You must have a working [Go](https://golang.org/doc/install) environment.

```console
$ git clone git@github.com:open-service-mesh/osm.git
$ cd osm
$ make build-osm
```
`make build-osm` will fetch any required dependencies, compile `osm` and place it in `bin/osm`. Add `bin/osm` to `$PATH` so you can easily use `osm`.

## Create Environment Variables
Create some necessary environment variables. This environment variable setup is a temporary step because OSM is currently a private project and there are no public container images available.
```console
$ # K8S_NAMESPACE is the Namespace the control plane will be installed into
$ export K8S_NAMESPACE=osm-system

$ # CTR_REGISTRY is the URL of the container registry to use
$ export CTR_REGISTRY=<your registry>

$ # If no authentication to push to the container registry is required, the following steps may be skipped.

$ # For Azure Container Registry (ACR), the following command may be used: az acr credential show -n <your_registry_name> --query "passwords[0].value" | tr -d '"'
$ export CTR_REGISTRY_PASSWORD=<your password>

$ # Create docker secret in Kubernetes Namespace using following script:
$ ./scripts/create-container-registry-creds.sh "$K8S_NAMESPACE"

```

## Build and push OSM images
Build and push images necessary to install OSM. This is also a temporary step because OSM is currently a private project and there are no public container images available.

```console
$ make docker-push-osm-controller
$ make docker-push-init
```

## Install OSM Control Plane
Use the `osm` CLI to install the OSM control plane on to a Kubernetes cluster.

Run `osm install`.
```console
# Install osm control plane components
$ osm install
OSM installed successfully in namespace [osm-system] with mesh name [osm]
```

By default, the control plane components are installed into a Kubernetes Namespace called `osm-system` and the control plane is given a unique identifier attribute `mesh-name` defaulted to `osm`. Both the Namespace and mesh-name can be configured with flags to the `osm install` command.

## Inspect Control Plane Components
A few components will be installed by defaut into the `osm-system` Namespace. Inspect them by using the following `kubectl` command:
```console
$ kubectl get pods,svc,secrets,configmaps,serviceaccount --namespace osm-system
```

A few cluster wide (non Namespaced components) will also be installed. Inspect them using the following `kubectl` command:
```console
kubectl get clusterrolebinding,clusterrole,mutatingwebhookconfiguration
```

Under the hood, `osm` is using [Helm](https://helm.sh) libraries to create a Helm `release` object in the control plane Namespace. The Helm `release` name is the mesh-name. The `helm` CLI can also be used to inspect Kubernetes manifests installed in more detail.
```console
$ helm get manifest osm --namespace osm-system
```
