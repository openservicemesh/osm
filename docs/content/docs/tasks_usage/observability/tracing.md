---
title: "Tracing"
description: "Tracing"
type: docs
aliases: ["tracing.md"]
---

# Tracing
Open Service Mesh (OSM) allows optional deployment of Jaeger for tracing. Similarly, tracing can be enabled and customized during installation (`tracing` section in `values.yaml`) or at runtime by editing the `osm-config` ConfigMap. Tracing can be enabled, disabled and configured at any time to support BYO scenarios.

## Jaeger
[Jaeger](https://www.jaegertracing.io/) is an open source tracing system used for monitoring and troubleshooting distributed systems. It can be deployed with OSM as a new instance or you may bring your own instance.

### Automatic bring up
By default, Jaeger deployment and tracing as a whole is disabled.

Jaeger can be automatically deployed by using the `--deploy-jaeger` OSM CLI flag at install time or by toggling the `deployJaeger` value in `values.yaml`.


### BYO (Bring-your-own)
The following section documents the additional steps needed to allow an already running instance of Jaeger to integrate with your OSM control plane.
> NOTE: This guide outlines steps specifically for Jaeger but you may use your own tracing application instance with applicable values

#### Prerequisites
* A running Jaeger instance

The sections below outline how to make required updates depending on whether you already have a running instance of OSM or are installing OSM for the first time. In either case, the following `tracing` values in `values.yaml` are being updated to point to your Jaeger instance:
1. `enable`: set to `true` to tell the Envoy connection manager to send tracing data to a specific address (cluster)
1. `address`: set to the destination cluster of your Jaeger instance
1. `port`: set to the destination port for the listener that you intend to use
1. `endpoint`: set to the destination's API or collector endpoint where the spans will be sent to


#### Deploying Jaeger to a running instance of the OSM control plane

If you already have OSM running, `tracing` values must be updated in the OSM ConfigMap using:

> Note: Replace `osm-system` in the commands below with osm-controller's namespace if using a non default namespace

1. Enable tracing.
```bash
kubectl patch ConfigMap osm-config \
  --namespace osm-system \
  --type=merge \
  --patch '{"data":{"tracing_enable": "true"}}'
```

2. Provide the **address** of the Jaeger instance to the OSM Controller. (Example: `jaeger.osm-system.svc.cluster.local`)
```bash
kubectl patch ConfigMap osm-config \
  --namespace osm-system \
  --type=merge \
  --patch '{"data":{"tracing_address": "<tracing server hostname>"}}'
```

3. Provide the **port** of the Jaeger instance to the OSM Controller. (Example: `9411`)
```bash
kubectl patch ConfigMap osm-config \
  --namespace osm-system \
  --type=merge \
  --patch '{"data":{"tracing_port": "<tracing server port>"}}'
```

4. Provide the **endpoint** of the Jaeger instance to the OSM Controller. (Example: `/api/v2/spans`)
```bash
kubectl patch ConfigMap osm-config \
  --namespace osm-system \
  --type=merge \
  --patch '{"data":{"tracing_endpoint": "<tracing server endpoint>"}}'
```


> Note: To make this change persistent between upgrades, see osm mesh upgrade --help.

You can verify these changes have been deployed by inspecting `osm-config`:
```bash
kubectl get configmap osm-config -n osm-system -o yaml
```

#### Deploying Jaeger during OSM deployment

To deploy _your own instance of Jaeger_ during OSM installation, you can use the `--set` flag as shown below to update the values:

```bash
osm install --set \
    OpenServiceMesh.tracing.enable=true, \
    OpenServiceMesh.tracing.address=<tracing server hostname>, \
    OpenServiceMesh.tracing.port=<tracing server port>, \
    OpenServiceMesh.tracing.endpoint=<tracing server endpoint>
```
