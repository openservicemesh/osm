---
title: "High-level Software Architecture"
description: "High-level Software Architecture"
type: docs
weight: 2
---

## High-level software architecture

The Open Service Mesh project is composed of the following five high-level components:

1. [Proxy control plane](#1-proxy-control-plane) - handles gRPC connections from the service mesh sidecar proxies
2. [Certificate manager](#2-certificate-manager) - handles issuance and management of certificates
3. [Endpoints providers](#3-endpoints-providers) - components capable of introspecting the participating compute platforms; these retrieve the IP addresses of the compute backing the services in the mesh
4. [Mesh specification](#4-mesh-specification) - wrapper around the [SMI Spec's Go SDK](https://github.com/deislabs/smi-sdk-go); this facility provides simple methods to retrieve [SMI Spec](https://smi-spec.io/) [resources](https://github.com/deislabs/smi-spec#service-mesh-interface), abstracting away cluster and storage specifics
5. [Mesh catalog](#5-mesh-catalog) - the service mesh's heart; this is the central component that collects inputs from all other components and dispatches configuration to the proxy control plane

![components relationship](https://user-images.githubusercontent.com/49918230/73027022-8b030180-3e2a-11ea-8226-e466b5a68e0c.png)

([source](https://microsoft-my.sharepoint.com/:p:/p/derayche/EZRZ-xXd06dFqlWJG5nn2wkBQCm8MMlAtRcNk6Yuir9XhA?e=zPw4FZ))
