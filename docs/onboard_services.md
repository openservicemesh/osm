# Onboard Services
The following guide describes how to onboard Kubernetes services to an OSM instance.


1. Configure and install [SMI policies](https://github.com/servicemeshinterface/smi-spec) to enable traffic to flow between services in the mesh.

    By default, OSM denies all traffic unless explicitly allowed by SMI policies. If this behavior is overridden with the `--enable-permissive-traffic-policy` flag on the `osm install` command, SMI policies are not required to allow traffic and services can still take advantage of features such as mTLS-encrypted traffic, metrics, and tracing.

    See [demo/deploy-traffic-spec.sh](/demo/deploy-traffic-spec.sh), [demo/deploy-traffic-split.sh](/demo/deploy-traffic-split.sh), and [demo/deploy-traffic-target.sh](/demo/deploy-traffic-target.sh) for examples.

1. Add namespaces containing services to the mesh with the `osm namespace add` command. All new pods created in added namespaces will automatically have a proxy sidecar container injected. Specific pods can be labeled to prevent sidecar injection. See [SIDECAR-INJECTION](SIDECAR-INJECTION.md) for more details.

    See [demo/join-namespaces.sh](/demo/join-namespaces.sh) for an example.

1. Restart the pods backing services in the mesh to inject the sidecars.

    Once manual sidecar injection is supported, this step will no longer be required.

    See [demo/rolling-restart.sh](/demo/rolling-restart.sh) for an example.

1. Verify the new behavior.

    The OSM control plane installs Prometheus and Grafana instances by default that can be used to help make sure the application is working properly. More details can be found in [OBSERVABILITY.md](OBSERVABILITY.md).


#### Note: Removing Namespaces
Namespaces can be removed from a mesh with the `osm namespace remove` command. Note that this command only tells OSM to stop applying updates to proxy configurations in the namespace. It does not remove the proxy sidecars, so the existing proxy configuration will continue to be used, but it will not be updated by the OSM control plane. To remove the proxies from all pods, remove the pods' namespaces with the CLI and reinstall all the pod workloads.
