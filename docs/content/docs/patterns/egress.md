---
title: "Egress"
description: "Enable access to the Internet and services external to the service mesh."
type: docs
---

# Allowing access to the Internet and out-of-mesh services (egress)

This document describes the steps required to enable access to the Internet and services external to the service mesh, sometimes referred to as `egress` traffic.

OSM redirects all outbound traffic from a pod within the mesh to the pod's sidecar proxy. Outbound traffic can be classified into two categories:

1. Traffic to services within the mesh cluster, referred to as in-mesh traffic
2. Traffic to services external to the mesh cluster, referred to as egress traffic

While in-mesh traffic is routed based on L7 traffic policies, egress traffic is routed differently and is not subject to in-mesh traffic policies. OSM supports access to external services as a passthrough without subjecting such traffic to filtering policies.


## Configuring Egress

Enabling egress is done via a global setting. The setting is toggled on or off and affects all services in the mesh. Egress is enabled by default when OSM is installed.

### Enabling egress
Egress can be enabled during OSM install or post install. When egress is enabled, outbound traffic from pods are allowed to egress the pod as long as the traffic does not match in-mesh traffic policies that otherwise deny the traffic.

Egress can be configured using either of the following ways.
1. During OSM install (default `--enable-egress=false`)
	```bash
	osm install --enable-egress
	```

2. Post OSM install

	`osm-controller` retrieves the egress configuration from the `osm-config` ConfigMap in its namespace (`osm-system` by default). Patch the ConfigMap by setting `egress: "true"`.
	```bash
	kubectl patch ConfigMap osm-config -n osm-system -p '{"data":{"egress":"true"}}' --type=merge
	```

### Disabling Egress

Similar to enabling egress, egress can be disabled during OSM install or post install.
1. During OSM install
	```bash
	bin osm install --enable-egress=false
	```

2. Post OSM install
	Patch the `osm-config` ConfigMap and set `egress: "false"`.
	```bash
	kubectl patch ConfigMap osm-config -n osm-system -p '{"data":{"egress":"false"}}' --type=merge
    ```

With egress disabled, traffic from pods within the mesh will not be able to access external services outside the cluster.
