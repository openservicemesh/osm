# Open Service Mesh Development Guide

Welcome to the Open Service Mesh development guide!

This document will help you build and run Open Service Mesh from source.
More information about running the demo included in this repo is
in [/demo/README.md](../demo/README.md).
The OSM software design is discussed
in detail in [DESIGN.md](/DESIGN.md).

## Table of contents

- [Repo Layout](#repo-layout)
- [Open Service Mesh Components](#openservicemesh-components)
- [Development Configurations](#development-configurations)
- [Helm Chart](#helm-chart)
- [Build Architecture](#build-architecture)


## Repo Layout

This in a non-exhaustive list of the directories in this repo. It is provided
as a birds-eye view of where the different components are located.

  - `charts/` - contains OSM Helm chart
  - `ci/` - tools and scripts for the continuos integration system
  - `cmd/` - OSM command line tools
  - `crd/` - Custom Resource Definitions needed by OSM
  - `demo/` - scripts and Kubernetes resources needed to run the Bookstore demonstration of Open Service Mesh
  - `docs/` - OSM documentation
  - `pkg/` -
    - `catalog/` - Mesh Catalog component is the central piece of OSM, which collects inputs from all other components and dispatches configuration to the proxy control plane
    - `certificate/` - contains multiple implementations of 1st and 3rd party certificate issuers, as well as PEM and x509 certificate management tools
        - `providers/` -
          - `keyvault/` - implements integration with Azure Key Vault
          - `vault/` - implements integration with Hashicorp Vault
          - `tresor/` - OSM native certificate issuer
    - `debugger/` - web server and tools used to debug the service mesh and the controller
    - `endpoint/` - Endpoints are components capable of introspecting the participating compute platforms; these retrieve the IP addresses of the compute backing the services in the mesh. This directory contains integrations with supported compute providers.
      - `providers/` -
        - `azure/` - integrates with Azure
        - `kube/` - Kubernetes tools and informers integrations
    - `envoy/` - packages needed to translate SMI into xDS
      - `ads/` - Aggregated Discovery Service related tools
      - `cds/` - Cluster Discovery Service related tools
      - `cla/` - Cluster Load Assignment components
      - `eds/` - Endpoint Discovery Service tools
      - `lds/` - Listener Discovery Service tools
      - `rds/` - Route Discovery Service tools
      - `sds/` - Secret Discovery service related tools
    - `health/` - OSM controller liveness and readiness probe handlers
    - `ingress/` - package mutating the service mesh in response to the application of an Ingress Kubernetes resource
    - `injector/` - sidecar injection webhook and related tools
    - `kubernetes/` - Kubernetes event handlers and helpers
    - `logger/` - logging facilities
    - `metricsstore/` - OSM controller system metrics tools
    - `namespace/` - package with tools handling a service mesh spanning multiple Kubernetes namespaces.
    - `osm_client/` - OSM client
    - `service/` - tools needed for easier handling of Kubernetes services
    - `signals/` - operating system signal handlers
    - `smi/` - SMI client, informer, caches and tools
    - `tests/` - test fixtures and other functions to make unit testing easier
    - `trafficpolicy/` - SMI related types


Open Service Mesh (OSM) is written in Go. It relies on the [SMI Spec](https://github.com/servicemeshinterface/smi-spec/) standard
and leverages [Envoy proxy](https://github.com/envoyproxy/envoy) as a data plane.
OSM uses Envoy's [go-control-plane (XDS)](https://github.com/envoyproxy/go-control-plane) package.



## Open Service Mesh components

- cli Command-line `osm` utility, view and drive the control
  plane.
- Helm chart
- controller


## Development Configurations

Depending on use case, there are several configurations with which to develop
and run Open Service Mesh.
The root of the repository contains a file named `.env.example`. Copy the contents of this file into `.env`: `cat .env.example > .env` and modify the contents of `.env` to suite your environment.

TODO:
 - Describe all the environment variables.
 - how to enable various levels of debugging - mostly tracing.
 - Describe the format of the debug messages.
 - How to enable the debug server & how it could be useful.

### Go

This repository uses Go v1.14.

#### Go modules and dependencies

This repo supports [Go Modules](https://github.com/golang/go/wiki/Modules).
The repo can be cloned outside of the `GOPATH`, since Go Modules support is
enabled by default since Go version 1.11.

If you are using this repo from within the `GOPATH`,
activate module support with: `export GO111MODULE=on`

#### Makefile

Many of the operations within the repo have GNU Makefile targets.
More notable:
  - `make build` builds the project
  - `make go-tests` to run unit tests
  - `make go-test-coverage` - run unit tests and output unit test coverage
  - `make go-lint` runs golint and golangci-lint
  - `make go-fmt` - same as `go fmt ./...`
  - `make go-vet` - same as `go vet ./...`

## Create Environment Variables
Some of the scripts and build targets available expect an accessible container registry where to push the `osm-controller` and `init` docker images once compiled. The location and credential options for the container registry can be specified as environment variables declared in `.env`, as well as the target namespace where `osm-controller` will be installed on.

Additionally, if using `demo/` scripts to deploy OSM's provided demo on your own K8s cluster, the same container registry configured in `.env` will be used to pull OSM images on your K8s cluster.
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
(NOTE: these requirements are true for automatic demo deployment using the available demo scripts, #1416 tracks an improvement to not strictly require these and use upstream images from official dockerhub registry if a user does not want/need changes on the code)

## Build and push OSM images
For development an/or testing locally compiled builds, pushing the local image to a container registry is still required. The following build targets will do so automatically against the configured container registry.

```console
$ make docker-push-osm-controller
$ make docker-push-init
```

#### Formatting

All Go source code is formatted with `goimports`. The version of `goimports`
used by this project is specified in `go.mod`. To ensure you have the same
version installed, run `go install -mod=readonly
golang.org/x/tools/cmd/goimports`. It's recommended that you set your IDE or
other development tools to use `goimports`. Formatting is checked during CI by
the `bin/fmt` script.


TODO
 - what about the golangci-lint ?
 - how to use the OSM cli
 - how do you deploy
 - how do you run CI
 - how do you run the demo

TODO: Explain how the git commit hash is embedded in the binary.

#### Docker

TODO:
  - what docker images are there - what they are
  - how we build and push to a public repository

## Helm chart

The Open Service Mesh control plane chart is located in the
[`charts/osm`](/charts/osm) folder.

The [`charts/osm/values.yaml`](/charts/osm/values.yaml) file defines the default value for properties
referenced by the different chart templates.

The [`charts/osm/templates/`](/charts/osm/templates/) folder contains the chart templates
for the different Kubernetes resources that are deployed as a part of the Open Service control plane installation.
The different chart templates are used as follows:
- `osm-*.yaml` chart templates are directly consumed by the `osm-controller` service.
- `mutatingwebhook.yaml` is used to deploy a `MutatingWebhookConfiguration` kubernetes resource that enables automatic sidecar injection
-  `grafana-*.yaml` chart templates are used to deploy a Grafana instance when the metrics stack is enabled
- `prometheus-*.yaml` chart templates are used to deploy a Prometheus instance when the metrics stack is enabled
- `zipkin-*.yaml` chart templates are used to deploy a Zipkin instance when Zipkin tracing is enabled

The [`charts/osm/crds/`](/charts/osm/crds/) folder contains the charts corresponding to the SMI CRDs.
Experimental CRDs can be found under [`charts/osm/crds/experimental/`](/charts/osm/crds/experimental/).


## Build & CI Architecture

[TODO]
