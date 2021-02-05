---
title: "OSM Components & Interations"
description: "OSM Components & Interations"
type: docs
weight: 1
---

## OSM Components & Interactions

![OSM Components & Interactions](./docs/content/images/osm-components-and-interactions.png)

### Containers

When a new Pod creation is initiated, OSM's
[MutatingWebhookConfiguration](https://github.com/openservicemesh/osm/blob/release-v0.3/charts/osm/templates/mutatingwebhook.yaml)
intercepts the
[create](https://github.com/openservicemesh/osm/blob/release-v0.3/pkg/injector/webhook.go#L295)
[pod](https://github.com/openservicemesh/osm/blob/release-v0.3/pkg/injector/webhook.go#L299)
operations for [namespaces joined to the mesh](https://github.com/openservicemesh/osm/blob/release-v0.3/charts/osm/templates/mutatingwebhook.yaml#L19),
and forwards these API calls to the
[OSM control plane](https://github.com/openservicemesh/osm/blob/release-v0.3/charts/osm/templates/mutatingwebhook.yaml#L11).
OSM control plane augments ([patches](https://github.com/openservicemesh/osm/blob/release-v0.3/pkg/injector/webhook.go#L202-L208))
the Pod spec with 2 new containers.
One is the [Envoy sidecar](https://github.com/openservicemesh/osm/blob/release-v0.3/pkg/injector/patch.go#L82-L86),
the other is an [init container](https://github.com/openservicemesh/osm/blob/release-v0.3/pkg/injector/patch.go#L61-L74).
The init container is ephemeral. It executes the [init-iptables.sh bash script](https://github.com/openservicemesh/osm/blob/release-v0.3/init-iptables.sh)
and terminates.
The init container requires [NET_ADMIN Kernel capability](https://github.com/openservicemesh/osm/blob/release-v0.3/pkg/injector/init-container.go#L21-L25) for
[iptables](https://en.wikipedia.org/wiki/Iptables) changes to be applied.
OSM uses `iptables` to ensure that all inbound and outbound traffic flows through the Envoy sidecar.
The [init container Docker image](https://hub.docker.com/r/openservicemesh/init)
is passed as a string pointing to a container registry. This is passed via the `--init-container-image` CLI param to the OSM controller on startup. The default value is defined in the [OSM Deployment chart](https://github.com/openservicemesh/osm/blob/release-v0.3/charts/osm/templates/osm-deployment.yaml#L33).
