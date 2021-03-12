---
title: "Application Requirements"
description: "Application Requirements"
type: docs
---

# Application Requirements

## Application UIDs
Do not run applications with the user ID (UID) value of **1500**. This is **reserved** for the Envoy proxy sidecar container injected into pods by OSM's sidecar injector.

## Ports
Do not use the following ports as they are used by the Envoy sidecar.
| Port  | Description |
| ------| ----------- |
| 15000 | Envoy Admin Port |
| 15001 | Envoy Outbound Listener Port |
| 15003 | Envoy Inbound Listener Port |
| 15010 | Envoy Prometheus Inbound Listener Port |