---
title: "Design & Concepts"
description: "Detailed design and architecture of Open Service Mesh."
type: docs
aliases: ["design & concepts"]
weight: 2
---

## Overview

Open Service Mesh (OSM) is a simple, complete, and standalone [service mesh](https://en.wikipedia.org/wiki/Service_mesh) solution.
OSM provides a fully featured control plane. It leverages an architecture based on [Envoy](https://www.envoyproxy.io/) reverse-proxy sidecar.
While by default OSM ships with Envoy, the design utilizes [interfaces](#interfaces), which enable integrations with any xDS compatible reverse-proxy.
OSM relies on [SMI Spec](https://smi-spec.io/) to reference services that will participate in the service mesh.
OSM ships out-of-the-box with all necessary components to deploy a complete service mesh spanning multiple compute platforms.

## Use Case

_As on operator of services spanning diverse compute platforms (Kubernetes and Virtual Machines on public and private clouds) I need an open-source solution, which will dynamically_:

- **Apply policies** governing TCP & HTTP access between peer services
- **Encrypt traffic** between services leveraging mTLS and short-lived certificates with a custom CA
- **Rotate certificates** as often as necessary to make these short-lived and remove the need for certificate revocation management
- **Collect traces and metrics** to provide visibility into the health and operation of the services
- **Implement traffic split** between various versions of the services deployed as defined via [SMI Spec](https://smi-spec.io/)

_The system must be:_

- easy to understand
- simple to install
- effortless to maintain
- painless to troubleshoot
- configurable via [SMI Spec](https://smi-spec.io/)
