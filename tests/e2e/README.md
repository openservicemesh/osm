# E2E OSM Testing
## Table of Contents
- [Overview](#overview)
- [Files and structure](#files-and-structure)
- [Running the tests](#running-the-tests)
  - [Kind cluster](#kind-cluster)
  - [Other K8s deployment](#other-k8s-deployment)
  - [Flags](#flags)

## Overview
End-to-end tests verify the behavior of the entire system. For OSM, e2e tests will install a control plane, install test workloads and SMI policies, and check that the workload is behaving as expected.

## Files and structure
OSM's e2e tests are located in `tests/e2e`.
The tests are written using Ginkgo and Gomega so they may also be directly invoked using `go test`. Be sure to build the `osm-controller` and `init` container images and `osm` CLI before directly invoking the tests ([see instructions below](#running-the-tests)).

OSM's framework, helpers and related files are located under `tests/framework`.
Once imported, it automatically sets up an init mechanism which will automatically initialize and parse flags and variables from both `env` and `go test flags` if any are passed to the test. The hooks for initialization and cleanup are set at Ginkgo's `BeforeEach` at the top level of test execution (between Ginkgo `Describes`); we henceforth recommend keeping every test in its own `Describe` section, as well as on a separate file for clarity. You can refer to [common.go](tests/framework/common.go) for more details about the init, setup and cleanup processes.

Tests are organized by top-level `Describe` blocks into tiers based on priority. A tier's tests will also be run as a part of all the tiers below it.

- Tier 1: run against every PR and should pass before being merged
- Tier 2: run against every merge into the main branch

Independent of tiers, tests are also organized into buckets. Each bucket runs in parallel, and individual tests in the bucket run sequentially.

**Note**: These tiers and buckets and which tests fall into each are likely to change as the test suite grows.

To help organize the tests, a custom `Describe` block named `OSMDescribe` is provided which accepts an additional struct parameter which contains fields for test metadata like tier and bucket. `OSMDescribe` will construct a well-formatted name including the test metadata which can be used in CI to run tests accordingly. Ginkgo's original `Describe` should not be used directly at the top-level and `OSMDescribe` should be used instead.

## Running the tests
Running the tests will require a running Kubernetes cluster. If you do not have a Kubernetes cluster to run the tests onto, you can choose to run them using `Kind`, which will make the test framework initialize a cluster on a local accessible docker client.

Running the tests will also require the [Helm](https://helm.sh/) CLI to be installed on your machine.

The tests can be run using the `test-e2e` Makefile target at repository root level (which defaults to use Kind), or alternatively `go test` targetting the test folder, which gives more flexibility but depends on related `env` flags given or parsed by the test.

Please refer to the [Kind cluster](#kind-cluster) or [Other K8s deployment](#other-k8s-eployment) and follow the instructions to setup potential env flags required by either option.

In addition to the flags provided by `go test` and Ginkgo, there are several custom command line flags that may be used for e2e tests to configure global parameters like container image locations and cleanup behavior. You can see the list of flags under the [flag section](#flags) below.

### Kind cluster
The following `make` target will create local containers for the OSM components, tagging them with `CTR_TAG`, and will launch the tests using Kind cluster. A Kind cluster is created at test start, and requires a docker interface to be available on the host running the test.
When using Kind, we load the images onto the Kind nodes directly (as opposed to providing a registry to pull the images from).
```
CTR_TAG=not-latest make test-e2e
```
Note: If you use `latest` tag, K8s will try to pull the image by default. If the images are not pushed to a registry accessible by the kind cluster, image pull errors will occur. Or, if an image with the same name is available, like `openservicemesh/init:latest`, then that publicly available image will be pulled and started instead, which may not be as up-to-date as the local image already loaded onto the cluster.

### Other K8s deployment
Have your Kubeconfig file point to your testing cluster of choice.
The following code uses `latest` tag by default. Non-Kind deployments do not push the images on the nodes, so make sure to set the registry accordingly.
```
export CTR_REGISTRY=<myacr>.dockerhub.io # if needed, set CTR_REGISTRY_USER and CTR_REGISTRY_PASSWORD
make build-osm
make docker-push
go test ./tests/e2e -test.v -ginkgo.v -ginkgo.progress
```

### Flags
#### (TODO) Kubeconf selection
Currently, test init will load a `Kubeconf` based on Defalut Kubeconf Loading rules.
If Kind is used, the kubeconf is temporarily replaced and Kind's kubeconf is used instead.

#### Container registry
A container registry where to load the images from (OSM, init container, etc.). Credentials are optional if the container registry allows pulling the images publicly:
```
-ctrRegistry string
		Container registry
-ctrRegistrySecret string
		Container registry secret
-ctrRegistryUser string
		Container registry username
```
If container registry user and password are provided, the test framework will take care to add those as Docker secret credentials for the given container registry whenever appropriate (tenant namespaces for `init` containers, OSM intallation, etc).
Container registry related flags can also be set through env:
```
export CTR_REGISTRY=<your_cr>.dockerhub.io
export CTR_REGISTRY_USER=<uername>             # opt
export CTR_REGISTRY_PASSWORD=<password>        # opt
```

#### OSM Tag
The following flag will refer to the image version of the OSM platform containers (`osm-controller` and `init`) and `tcp-echo-server` for the tests to use:
```
-osmImageTag string
		OSM image tag (default "latest")
```
Make sure you have compiled the images and pushed them on your registry first if you are not using a kind cluster:
```
export CTR_REGISTRY=myacr.dockerhub.io
export CTR_TAG=mytag               # Optional, 'latest' used by default
make docker-push-init docker-push-osm-controller.    # Use docker-build-* targets instead when using kind
```

#### Use Kind for testing
Testing implements support for Kind. If `kindCluster` is enabled, a new Kind cluster will be provisioned and it will be automatically used for the test.
```
-kindCluster
		Creates kind cluster
-kindClusterName string
		Name of the Kind cluster to be created (default "osm-e2e")
-cleanupKindCluster
		Cleanup kind cluster upon exit (default true)
-cleanupKindClusterBetweenTests
		Cleanup kind cluster between tests (default true)
```

#### Test specific flags

Worth mentioning `cleanupTest` is especially useful for debugging or leaving the test in a certain state at test-exit.
When using Kind, you need to use `cleanupKindCluster` and `cleanupKindClusterBetweenTests` in conjunction, or else the cluster
will anyway be destroyed.
```
-cleanupTest
		Cleanup test resources when done (default true)
-meshName string
		OSM mesh name (default "osm-system")
-waitForCleanup
		Wait for effective deletion of resources (default true)
```
Plus, `go test` and `Ginkgo` specific flags, of course.
