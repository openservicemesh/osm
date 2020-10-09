# E2E OSM Testing
End-to-end tests verify the behavior of the entire system. For OSM, e2e tests will install a control plane, install test workloads and SMI policies, and check that the workload is behaving as expected. 

## Files and structure
OSM's e2e tests are located in tests/e2e. The tests can be run using the `test-e2e` Makefile target. The Makefile target will also build the necessary container images and `osm` CLI binary before running the tests. The tests are written using Ginkgo and Gomega so they may also be directly invoked using `go test`. Be sure to build the `osm-controller` and `init` container images and `osm` CLI before directly invoking the tests. With either option, it is suggested to explicitly set the container registry location and tag to ensure up-to-date images are used by setting the `CTR_REGISTRY` and `CTR_TAG` environment variables.

In addition to the flags provided by `go test` and Ginkgo, there are several custom command line flags that may be used for e2e tests to configure global parameters like container image locations and cleanup behavior. You can see the list of flags and explanatory use down below this document.

The hooks for initialization and cleanup are set at Ginkgo's `BeforeEach` at the top level of test execution (between  Ginkgo `Describes`); we henceforth recommend keeping every test in its own `Describe` section, as well as on a separate file for clarity. You can refer to `suite_test.go` for more details.

## Quick Start
### Kind cluster
The following `make` target will create local containers for the OSM components, tagging them with `CTR_TAG`, and will launch the tests using Kind cluster. A Kind cluster is created at test start, and requires a docker interface to be available on the host running the test.
When using Kind, we load the images onto the Kind nodes directly (instead of providing a registry).
```
CTR_TAG=not-latest make test-e2e
```
Note: If you use `latest` tag, K8s will try to pull the image by default. If the images are not pushed to a registry accessible by the kind cluster, image pull errors will occur. Or, if an image with the same name is available, like `openservicemesh/init:latest`, then that publicly available image will be pulled and started instead, which may not be as up-to-date as the local image already loaded onto the cluster.

### Any other K8s deployment
Have your `Kubeconf` pointing to your testing cluster of choice.
The following code uses `latest` tag by default. Non-Kind deployments do not push the images on the nodes, so make sure to set the registry accordingly.
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

### Container registry
A container registry where to load the images from (OSM, init container, etc.). Credentials are optional if the container registry allows pulling the images publicly:
```
-ctrRegistry string
		Container registry
-ctrRegistrySecret string
		Container registry secret
-ctrRegistryUser string
		Container registry username
```
If CR user and password are provided, the test framework will take care to add those as Docker secret credentials for the given container registry whenever appropriate (tenant namespaces for `init` containers, OSM intallation, etc).
Container registry related flags can also be set through env:
```
export CTR_REGISTRY=<your_cr>.dockerhub.io
export CTR_REGISTRY_USER=<uername>             # opt
export CTR_REGISTRY_PASSWORD=<password>        # opt
```

### OSM Tag
The following flag will refer to the version of the OSM platform containers (OSM and init) for test to use:
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
