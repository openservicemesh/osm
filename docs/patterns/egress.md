# Allowing access to the Internet and out-of-mesh services (egress)

This document describes the steps required to enable access to the Internet and services external to the service mesh, sometimes referred to as `egress` traffic.

OSM redirects all outbound traffic from a pod within the mesh to the pod's sidecar proxy. Outbound traffic can be classified into two categories:

1. Traffic to services within the mesh cluster, referred to as in-mesh traffic
2. Traffic to services external to the mesh cluster, referred to as egress traffic

While in-mesh traffic is routed based on L7 traffic policies, egress traffic is routed differently and is not subject to in-mesh traffic policies. OSM supports access to external services as a passthrough without subjecting such traffic to filtering policies.


## Configuring Egress

Enabling egress is done via a global setting. The setting is toggled on or off and affects all services in the mesh. Egress is enabled by default when OSM is installed.

### Enabling egress
Egress can be enabled during OSM install or post install. When egress is enabled, OSM requires the mesh [CIDR](https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing) ranges to be specified. The mesh CIDR ranges are the list of CIDR ranges corresponding to the pod and service CIDRs configured in the cluster. The mesh CIDR ranges are required with egress to prevent any traffic destined within the cluster from escaping out as egress traffic, to be able to enforce mesh traffic policies.

A [convenience script](https://github.com/openservicemesh/osm/blob/main/scripts/get_mesh_cidr.sh) to retrieve the mesh CIDR ranges can be used if the user is not aware of the pod and service CIDR ranges for their cluster.
```console
$ ./scripts/get_mesh_cidr.sh
10.0.0.0/16,10.2.0.0/16
```

Egress can be configured using either of the following ways.
1. During OSM install (default `--enable-egress=false`)
	```bash
	osm install --enable-egress --mesh-cidr "10.0.0.0/16,10.2.0.0/16"
	```
	or
	```bash
	osm install --enable-egress --mesh-cidr 10.0.0.0/16 --mesh-cidr 10.2.0.0/16
	```

2. Post OSM install

	`osm-controller` retrieves the egress configuration from the `osm-config` ConfigMap in its namespace (`osm-system` by default). Patch the ConfigMap by setting `egress: "true"` and `mesh_cidr_ranges` with the CIDR ranges obtained above.
	```bash
	kubectl patch ConfigMap osm-config -n osm-system -p '{"data":{"egress":"true", "mesh_cidr_ranges":"10.0.0.0/16,10.2.0.0/16"}}' --type=merge
	```
	*Note: The value for `mesh_cidr_ranges` can either be space or comma separated.*


With egress enabled, traffic from pods within the mesh will be allowed to access external services outside the mesh CIDR ranges.

### Disabling Egress

Similar to enabling egress, egress can be disabled during OSM install or post install. The mesh CIDR ranges are not required when egress is being disabled.

1. During OSM install
	```bash
	bin osm install --enable-egress=false
	```

2. Post OSM install
	Patch the `osm-config` ConfigMap and set `egress: "false"`.
	```bash
	kubectl patch ConfigMap osm-config -n osm-system -p '{"data":{"egress":"false"}}' --type=merge
    ```

With egress disabled, traffic from pods within the mesh will not be able to access external services outside the mesh CIDR ranges.
