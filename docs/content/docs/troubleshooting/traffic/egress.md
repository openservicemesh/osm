---
title: "Egress Troubleshooting"
description: "Egress Troubleshooting Guide"
type: docs
aliases: ["egress.md"]
---

## When Egress is not working as expected

### 1. Confirm egress is enabled

Confirm egress is enabled by verifying the value for the `egress` key in the `osm-config` ConfigMap. `osm-config` ConfigMap resides in the namespace OSM control plane namespace, `osm-system` by default.

```console
# Returns true if egress is enabled
$ kubectl get configmap -n osm-system osm-config -o jsonpath='{.data.egress}{"\n"}'
true
```

The above command must return a boolean string (`true` or `false`) indicating if egress is enabled.

### 2. Inspect OSM controller logs for errors

```bash
# When osm-controller is deployed in the osm-system namespace
kubectl logs -n osm-system $(kubectl get pod -n osm-system -l app=osm-controller -o jsonpath='{.items[0].metadata.name}')
```

Errors will be logged with the `level` key in the log message set to `error`:
```console
{"level":"error","component":"...","time":"...","file":"...","message":"..."}
```

### 3. Confirm the Envoy configuration

Confirm the Envoy proxy configuration on the client has a default egress filter chain on the outbound listener. Refer to the [sample configurations](https://github.com/openservicemesh/osm/blob/main/docs/content/docs/tasks_usage/traffic_management/egress.md#envoy-configurations) to verify that the client is configured to have outbound access to external destinations.

## When the setting needs to be persisted across upgrades

While the `osm-config` ConfigMap can be directly updated using the `kubectl patch` command, to persist configuration changes across upgrades, it is recommended to always use `osm mesh upgrade` CLI command to update the mesh configuration.

Refer to the [configuring egress](https://github.com/openservicemesh/osm/blob/main/docs/content/docs/tasks_usage/traffic_management/egress.md#configuring-egress) section to enable or disable egress.