# E2E OSM Testing
This folder contains End-to-End (E2E) OSM tests.

E2E testing is mostly motivated by use-case driven scenarios, starting from a blank or empty cluster (or not even a cluster if Kind is backing the test, see details below), installing OSM from scratch, deploying applications and policies and ultimately testing traffic between specific cluster members to verify policy enforcement and proper mesh functioning. 

## Files and structure
Tests have been written under Ginkgo framework, and most of the helpers, wrappers and accessibility functions are provided in respective `common_*.go`, based on their areas of effect.

The framework exposes an OSM test data structure or handle, which is in effect the interaction mechanism for the test itself.  The test framework takes care to collect and initialize most of the common functionalities a test would expect when deploying on K8s, including but not limited to Kubernetes and SMI clientsets, flag parsing, container registry values, cleanup hooks, etc., and provides accessibility functions through the handle for the test to use at its own discretion.

The hooks for initialization and cleanup are set at Ginkgo's `BeforeEach` at the top level of test execution (between  Ginkgo `Describes`); we henceforth recommend keeping every test in its own `Describe` section, as well as on a separate file for clarity. You can refer to `suite_test.go` for more details.

## Quick Start
### Kind cluster
The following `make` target will create local containers for the OSM components, tagging them with `CTR_TAG`, and will launch the tests using Kind cluster. A kind cluster is created at test start, and requires a docker interface to be available on the host running the test.
When using Kind, we push the `CTR_TAG` images to the Kind nodes (instead of providing a registry).
```
CTR_TAG=not-latest make test-e2e
```
Note: If you use `latest` tag, and have not defined a registry, K8s will, by default, check the images on dockerhub, pulling and overriding the locally pushed images on the node.

### Any other K8s deployment
Have your `Kubeconf` pointing to your testing cluster of choice.
The following code uses `latest` tag by default.
```
export CTR_REGISTRY=<myacr>.dockerhub.io # if needed, set CTR_REGISTRY_USER and CTR_REGISTRY_PASSWORD 
make build-osm
make docker-push
go test ./tests/e2e -test.v -ginkgo.v -ginkgo.progress
```

## Flags
### (TODO) Kubeconf selection
Currently, test init will load a `Kubeconf` based on Defalut Kubeconf Loading rules. 
If Kind is used, the kubeconf is temporarily replaced and Kind's kubeconf is used instead.

### Container registry (required)
A container registry where to load the images from (OSM, init container, etc.) is necessary. Can be set by flag or env value:
```
-ctrRegistry string
		Container registry
-ctrRegistrySecret string
		Container registry secret
-ctrRegistryUser string
		Container registry username
```
If CR user and password are provided, the test framework will take care to add those as Docker secret credentials for the given container registry whenever appropriate (tenant namespaces for `init` containers, OSM intallation, etc).
You can also set them through env:
```
export CTR_REGISTRY=<your_cr>.dockerhub.io
export CTR_REGISTRY_USER=<uername>             # opt
export CTR_REGISTRY_PASSWORD=<password>        # opt
```

### OSM Tag
It will refer to the version of the OSM platform containers (OSM and init) for test to use:
```
-osmImageTag string
		OSM image tag (default "latest")
```
Make sure you have compiled the images and pushed them on your registry first:
```
export CTR_REGISTRY=myacr.dockerhub.io
export CTR_TAG=mytag               # Optional, 'latest' used by default
make docker-push
```

### Use Kind for testing 
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

### Test specific flags

Worth mentioning `cleanupTest` is specially useful for debugging or leaving the test in a certain state at test-exit.
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
