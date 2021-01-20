---
title: "Development Guide"
description: "Open Service Mesh Development Guide"
type: docs
---

# Open Service Mesh Development Guide

Welcome to the Open Service Mesh development guide!
Thank you for joining us on a journey to build an SMI-native lightweight service mesh. The first of our [core principles](https://github.com/openservicemesh/osm#core-principles) is to create a system, which is "simple to understand and contribute to." We hope that you would find the source code easy to understand. If not - we invite you to help us fulfill this principle. There is no PR too small!

To understand *what* Open Service Mesh does - take it for a spin and kick the tires. Install it on your Kubernetes cluster by following [this guide](./example/README.md).

To get a deeper understanding of how OSM functions - take a look at the detailed [software design](../design/).

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
    - `service/` - tools needed for easier handling of Kubernetes services
    - `signals/` - operating system signal handlers
    - `smi/` - SMI client, informer, caches and tools
    - `tests/` - test fixtures and other functions to make unit testing easier
    - `trafficpolicy/` - SMI related types
</details>

The Open Service Mesh controller is written in Go.
It relies on the [SMI Spec](https://github.com/servicemeshinterface/smi-spec/).
OSM leverages [Envoy proxy](https://github.com/envoyproxy/envoy) as a data plane and Envoy's [XDS v3](https://www.envoyproxy.io/docs/envoy/latest/api-v3/api) protocol, which is offered in Go by [go-control-plane](https://github.com/envoyproxy/go-control-plane).


## Get Go-ing

This repository uses [Go v1.15](https://golang.org/). If you are not familiar with Go, spend some time with the excellent [Tour of Go](https://tour.golang.org/).

## Get the dependencies

The OSM packages rely on many external Go libraries.

Take a peek at the [go.mod](https://github.com/openservicemesh/osm/blob/main/go.mod) file to see all dependencies.

Run `go get -d ./...` to download all required Go packages.

#### Makefile

Many of the operations within the repo have GNU Makefile targets.
More notable:
  - `make build` builds the project
  - `make go-test` to run unit tests
  - `make go-test-coverage` - run unit tests and output unit test coverage
  - `make go-lint` runs golangci-lint
  - `make go-fmt` - same as `go fmt ./...`
  - `make go-vet` - same as `go vet ./...`

## Create Environment Variables

The OSM repo relies on environment variables to make it usable on your localhost. The root of the repository contains a file named `.env.example`. Copy the contents of this file into `.env`
```bash
cat .env.example > .env
```
The various environment variables are documented in the `.env.example` file itself. Modify the variables in `.env` to suite your environment.

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

## Testing your changes

The OSM repo has a few layers of tests:
  - unit tests
  - integration tests
  - simulations

For tests in this repo we have chosen to leverage the
[Gomega](https://onsi.github.io/gomega/) and
[Ginkgo](https://onsi.github.io/ginkgo/) frameworks. We follow Go's convention and add
unit tests for the respective functions in files with the `_test.go` suffix. So if a
function lives in a file `foo.go` we will write a test for it in the file `foo_test.go`.
For more about Go testing
read [the following document](https://tip.golang.org/cmd/go/#hdr-Test_packages).

Take a look at any of the [existing unit-test examples](https://github.com/openservicemesh/osm/blob/release-v0.2/pkg/catalog/catalog_test.go)
should you need a starting point.

Often times we add a [suite_test.go](https://github.com/openservicemesh/osm/blob/release-v0.2/pkg/catalog/suite_test.go)
file, which serves as an entry point for the Ginkgo tests within the given package.

#### Unit Tests
The most rudimentary tests are the unit tests. We strive for test coverage above 80% where
this is pragmatic and possible.
Each newly added function should be accompanied by a unit test. Ideally, while working
on this repository, we practice
[test-driven development](https://en.wikipedia.org/wiki/Test-driven_development),
and each change would be accompanied by a unit test.

To run all unit tests you can use the following `Makefile` target:
```bash
make go-tests
```

You can run the tests exclusively for the package you are working on. For example the following command will
run only the tests in the package implementing OSM's
[Hashicorp Vault](https://www.vaultproject.io/) integration:
```bash
go test ./pkg/certificate/providers/vault/...
```

You can check the unit test coverage by using the `-cover` option:
```bash
go test -cover ./pkg/certificate/providers/vault/...
```

We have a dedicated tool for in-depth analysis of the unit-test code coverage:
```bash
./scripts/test-w-coverage.sh
```
Running the [test-w-coverage.sh](https://github.com/openservicemesh/osm/blob/main/scripts/test-w-coverage.sh) script will create
an HTML file with in-depth analysis of unit-test coverage per package, per
function, and it will even show lines of code that need work. Open the HTML
file this tool generates to understand how to improve test coverage:
```
open ./coverage/index.html
```

Once the file loads in your browser, scroll to the package you worked on to see current test coverage:

![package coverage](../images/unit-test-coverage-1.png)

Our overall guiding principle is to maintain unit-test coverage at or above 80%.

To understand which particular functions need more testing - scroll further in the report:

![per function](../images/unit-test-coverage-2.png)

And if you are wondering why a function, which we have written a test for, is not 100% covered,
you will find the per-function analysis useful. This will show you code paths that are not tested.

![per function](../images/unit-test-coverage-3.png)

##### Mocking

OSM uses the [GoMock](https://github.com/golang/mock) mocking framework to mock interfaces in unit tests.
GoMock's `mockgen` tool is used to autogenerate mocks from interfaces.

As an example, to create a mock client for the `Configurator` interface defined in the [configurator](https://github.com/openservicemesh/osm/tree/main/pkg/configurator) package:
```bash
go run github.com/golang/mock/mockgen -destination=pkg/configurator/mock_client.go -package=configurator github.com/openservicemesh/osm/pkg/configurator Configurator
```

When a mocked interface is changed, the autogenerated mock code must be regenerated.
More details can be found in [GoMock's documentation](https://github.com/golang/mock/blob/master/README.md).

#### Integration Tests

Unit tests focus on a single function. These ensure that with a specific input, the function
in question produces expected output or side effect. Integration tests, on the other hand,
ensure that multiple functions work together correctly. Integration tests ensure your new
code composes with other existing pieces.

Take a look at [the following test](https://github.com/openservicemesh/osm/blob/release-v0.2/pkg/configurator/client_test.go),
which tests the functionality of multiple functions together. In this particular example, the test:
  - uses a mock Kubernetes client via `testclient.NewSimpleClientset()` from the `k8s.io/client-go/kubernetes/fake` library
  - [creates a ConfigMap](https://github.com/openservicemesh/osm/blob/release-v0.2/pkg/configurator/client_test.go#L32)
  - [tests whether](https://github.com/openservicemesh/osm/blob/release-v0.2/pkg/configurator/client_test.go#L95-L96) the underlying functions compose correctly by fetching the results of the top-level function `GetMeshCIDRRanges()`

### End-to-End (e2e) Tests

End-to-end tests verify the behavior of the entire system. For OSM, e2e tests will install a control plane, install test workloads and SMI policies, and check that the workload is behaving as expected.

OSM's e2e tests are located in tests/e2e. The tests can be run using the `test-e2e` Makefile target. The Makefile target will also build the necessary container images and `osm` CLI binary before running the tests. The tests are written using Ginkgo and Gomega so they may also be directly invoked using `go test`. Be sure to build the `osm-controller` and `init` container images and `osm` CLI before directly invoking the tests. With either option, it is suggested to explicitly set the container registry location and tag to ensure up-to-date images are used by setting the `CTR_REGISTRY` and `CTR_TAG` environment variables.

In addition to the flags provided by `go test` and Ginkgo, there are several custom command line flags that may be used for e2e tests to configure global parameters like container image locations and cleanup behavior. The full list of custom flags can be found in [tests/e2e/](https://github.com/openservicemesh/osm/tree/main/tests/e2e#flags).

For more information, please refer to [OSM's E2E Readme](https://github.com/openservicemesh/osm/tree/main/tests/README.md).

#### Simulation / Demo
When we want to ensure that the entire system works correctly over time and
transitions state as expected - we run
[the demo included in the docs](https://github.com/openservicemesh/osm/blob/main/docs/example/README.md).
This type of test is the slowest, but also most comprehensive. This test will ensure that your changes
work with a real Kubernetes cluster, with real SMI policy, and real functions - no mocked or fake Go objects.

#### Profiling
OSM control plane exposes an HTTP server able to serve a number of resources.

For mesh visibility and debugabbility, one can refer to the endpoints provided under [pkg/debugger](https://github.com/openservicemesh/osm/tree/main/pkg/debugger) which contains a number of endpoints able to inspect and list most of the common structures used by the control plane at runtime.

Additionally, the current implementation of the debugger imports and hooks [pprof endpoints](https://golang.org/pkg/net/http/pprof/).
Pprof is a golang package able to provide profiling information at runtime through HTTP protocol to a connecting client.

Debugging endpoints can be turned on or off through the runtime argument `enable-debug-server`, normally set on the deployment at install time through the CLI.

Example usage:
```
scripts/port-forward-osm-debug.sh &
go tool pprof http://localhost:9091/debug/pprof/heap
```
From pprof tool, it is possible to extract a large variety of profiling information, from heap and cpu profiling, to goroutine blocking, mutex profiling or execution tracing. We suggest to refer to their [original documentation](https://golang.org/pkg/net/http/pprof/) for more information.

## Helm charts

The Open Service Mesh control plane chart is located in the
[`charts/osm`](https://github.com/openservicemesh/osm/tree/main/charts/osm) folder.

The [`charts/osm/values.yaml`](https://github.com/openservicemesh/osm/blob/main/charts/osm/values.yaml) file defines the default value for properties
referenced by the different chart templates.

The [`charts/osm/templates/`](https://github.com/openservicemesh/osm/tree/main/charts/osm/templates) folder contains the chart templates
for the different Kubernetes resources that are deployed as a part of the Open Service control plane installation.
The different chart templates are used as follows:
- `osm-*.yaml` chart templates are directly consumed by the `osm-controller` service.
- `mutatingwebhook.yaml` is used to deploy a `MutatingWebhookConfiguration` kubernetes resource that enables automatic sidecar injection
-  `grafana-*.yaml` chart templates are used to deploy a Grafana instance when grafana installation is enabled
- `prometheus-*.yaml` chart templates are used to deploy a Prometheus instance when prometheus installation is enabled
- `fluentbit-configmap.yaml` is used to provide configurations for the fluent bit sidecar and its plugins when fluent bit is enabled
- `jaeger-*.yaml` chart templates are used to deploy a Jaeger instance when Jaeger deployment and tracing are enabled

### Custom Resource Definitions

The [`charts/osm/crds/`](https://github.com/openservicemesh/osm/tree/main/charts/osm/crds/) folder contains the charts corresponding to the SMI CRDs.
Experimental CRDs can be found under [`charts/osm/crds/experimental/`](https://github.com/openservicemesh/osm/tree/main/charts/osm/crds/experimental).
