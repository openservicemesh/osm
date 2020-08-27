# Onboard Services
The following guide describes how to onboard a Kubernetes microservice to an OSM instance.


1. Configure and Install [Service Mesh Interface (SMI) policies](https://github.com/servicemeshinterface/smi-spec)

    OSM conforms to the SMI specification. By default, OSM denies all traffic communications between Kubernetes services unless explicitly allowed by SMI policies. This behavior can be overridden with the `--enable-permissive-traffic-policy` flag on the `osm install` command, allowing SMI policies not to be enforced while allowing traffic and services to still take advantage of features such as mTLS-encrypted traffic, metrics, and tracing.

    For example SMI policies, please see the following examples:
    - [demo/deploy-traffic-specs.sh](/demo/deploy-traffic-specs.sh)
    - [demo/deploy-traffic-split.sh](/demo/deploy-traffic-split.sh)
    - [demo/deploy-traffic-target.sh](/demo/deploy-traffic-target.sh)

1. Onboard Kubernetes Namespaces to enable OSM

    To onboard a namespace containing services to enable for OSM, run the `osm namespace add` command, which does the equivalent of the following:

    ```console
    $ kubectl label namespace <namespace> openservicemesh.io/monitored-by=<mesh-name>
    ```

    By default, the `osm namespace add` command does not enable the namespace for automatic sidecar injection. To enable automatic sidecar injection as a part of enrolling a namespace into the mesh, use `osm namespace add <namespace> --enable-sidecar-injection`. This does the equivalent of the following:

    ```console
    $ kubectl label namespace <namespace> openservicemesh.io/monitored-by=<mesh-name>
    $ kubectl annotate namespace <namespace> openservicemesh.io/sidecar-injection=enabled
    ```

    Once a namespace has been onboarded, pods can be enrolled in the mesh by configuring automatic sidecar injection. See the [Sidecar Injection](patterns/sidecar_injection.md) document for more details.

    For an example on how to onboard and join namespaces to the OSM mesh, please see the following example:
    - [demo/join-namespaces.sh](/demo/join-namespaces.sh)

1.  Inject the Proxy Sidecars

    At the moment to onboard your Kubernetes services to OSM, a restart of the pods backing the services is needed. In the near future, manual sidecar injection will be supported no longer requiring this step.

    For an example on how to invoke a rolling restart of your services' pods, please see the following example:
    - [demo/rolling-restart.sh](/demo/rolling-restart.sh)

1. Verify the new behavior

    The OSM control plane installs Prometheus and Grafana instances by default that can be used to help make sure the application is working properly. More details can be found in the [Observability](patterns/observability.md) document.


#### Note: Removing Namespaces
Namespaces can be removed from the OSM mesh with the `osm namespace remove` command, which does the equivalent of the following:

```console
$ kubectl label namespace <namespace> openservicemesh.io/monitored-by-
```

> **Please Note:**
> The **`osm namespace remove`** command only tells OSM to stop applying updates to the sidecar proxy configurations in the namespace. It **does not** remove the proxy sidecars. This means the existing proxy configuration will continue to be used, but it will not be updated by the OSM control plane. If you wish to remove the proxies from all pods, remove the pods' namespaces from the OSM mesh with the CLI and reinstall all the pod workloads.
