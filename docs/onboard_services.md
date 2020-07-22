# Onboard Services
The following guide describes how to onboard Kubernetes services to an OSM instance.


1. Configure and install [SMI policies](https://github.com/servicemeshinterface/smi-spec) to enable traffic to flow between services in the mesh.

    By default, OSM denies all traffic unless explicitly allowed by SMI policies. If this behavior is overridden with the `--enable-permissive-traffic-policy` flag on the `osm install` command, SMI policies are not required to allow traffic and services can still take advantage of features such as mTLS-encrypted traffic, metrics, and tracing.

    See [demo/deploy-traffic-spec.sh](/demo/deploy-traffic-spec.sh), [demo/deploy-traffic-split.sh](/demo/deploy-traffic-split.sh), and [demo/deploy-traffic-target.sh](/demo/deploy-traffic-target.sh) for examples.

1. Add namespaces containing services to the mesh with the `osm namespace add` command to enable automatic proxy sidecar injection.

    See [demo/join-namespaces.sh](/demo/join-namespaces.sh) for an example.

1. Restart the pods backing services in the mesh to inject the sidecars.

    See [demo/rolling-restart.sh](/demo/rolling-restart.sh) for an example.

1. Verify the new behavior.

    The OSM control plane installs Prometheus and Grafana instances by default that can be used to help make sure the application is working properly. More details can be found in [OBSERVABILITY.md](OBSERVABILITY.md).
