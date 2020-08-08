# Open Service Mesh Development Guide

Welcome to the Open Service Mesh development guide!
Thank you for joining us on a journey to build an SMI-native lightweight service mesh. The first of our [core principles](https://github.com/openservicemesh/osm#core-principles) is to create a system, which is "simple to understand and contribute to." We hope that you would find the source code easy to understand. If not - we invite you to help us fulfill this principle. There is no PR too small!

To understand *what* Open Service Mesh does - take it for a spin and kick the tires. Install it on your Kubernetes cluster by following [this guide](./example/README.md).

To get a deeper understanding of how OSM does what it does - take a look at the detailed [software design](../DESIGN.md).

When you are ready to jump in - [fork the repo](https://docs.github.com/en/github/getting-started-with-github/fork-a-repo) and then [clone it](https://docs.github.com/en/github/creating-cloning-and-archiving-repositories/cloning-a-repository) on your workstation.

The directories in the cloned repo will be structured approximately like this:
<details>
  <summary>Click to expand directory structure</summary>
This in a non-exhaustive list of the directories in this repo. It is provided
as a birds-eye view of where the different components are located.

  - `charts/` - contains OSM Helm chart
  - `ci/` - tools and scripts for the continuous integration system
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
</details>

The Open Service Mesh controller is written in Go.
It relies on the [SMI Spec](https://github.com/servicemeshinterface/smi-spec/).
OSM leverages [Envoy proxy](https://github.com/envoyproxy/envoy) as a data plane and Envoy's [XDS v3](https://github.com/envoyproxy/go-control-plane) protocol.


## Get Go-ing

This repository uses [Go v1.14](https://golang.org/). If you are not familiar with Go, spend some time with the excellent [Tour of Go](https://tour.golang.org/).

## Get the dependencies

The OSM packages rely on many external Go libraries.

Take a peek at the [go.mod](../go.mod) file to see all dependencies.

Run `go get -d ./...` to download all required Go packages.

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

The OSM repo relies on environment variables to make it usable on your localhost. The root of the repository contains a file named `.env.example`. Copy the contents of this file into `.env`
```bash
cat .env.example > .env
```
Tha various envirnoment variables are documented in the `.env` file itself. Modify the variables in `.env` to suite your environment.

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
make docker-push-osm-controller
make docker-push-init
```

## Code Formatting

All Go source code is formatted with `goimports`. The version of `goimports`
used by this project is specified in `go.mod`. To ensure you have the same
version installed, run `go install -mod=readonly
golang.org/x/tools/cmd/goimports`. It's recommended that you set your IDE or
other development tools to use `goimports`. Formatting is checked during CI by
the `bin/fmt` script.

## Helm charts

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

## Custom Resource Definitions

The [`charts/osm/crds/`](/charts/osm/crds/) folder contains the charts corresponding to the SMI CRDs.
Experimental CRDs can be found under [`charts/osm/crds/experimental/`](/charts/osm/crds/experimental/).
