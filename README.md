# Open Service Mesh (OSM)

[![build](https://github.com/open-service-mesh/osm/workflows/Go/badge.svg)](https://github.com/open-service-mesh/osm/actions?query=workflow%3AGo)
[![report](https://goreportcard.com/badge/github.com/open-service-mesh/osm)](https://goreportcard.com/report/github.com/open-service-mesh/osm)
[![codecov](https://codecov.io/gh/open-service-mesh/osm/branch/master/graph/badge.svg)](https://codecov.io/gh/open-service-mesh/osm)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/open-service-mesh/osm/blob/master/LICENSE)
[![release](https://img.shields.io/github/release/open-service-mesh/osm/all.svg)](https://github.com/open-service-mesh/osm/releases)

Open Service Mesh (OSM) is a lightweight, extensible, Cloud Native [service mesh][1] that allows users to uniformly manage, secure, and get out-of-the-box observability features for highly dynamic microservice environments.

Guided by 4 core principles:
1. Simple to understand and contribute to
1. Effortless to install, maintain, and operate
1. Painless to troubleshoot
1. Easy to configure via [SMI][2]

## Overview

OSM runs an Envoy based control plane on Kubernetes, can be configured with SMI APIs and works by injecting an Envoy proxy as a sidecar container next to each instance of your application. The proxy contains and executes rules around access control policies, implements routing configuration, and captures metrics. The control plane continually configures proxies to ensure policies and routing rules are up to date and ensures proxies are healthy.

Features of OSM:
1. More easily and transparently configure traffic shifting for deployments
1. Secure service to service communication by enabling mTLS
1. Define and execute fine grained access control policies for services
1. Observability and insights into application metrics for debugging and monitoring services
1. Integrate with external certificate management services/solutions with a pluggable interface.
1. Onboard applications onto the mesh by enabling automatic sidecar injection of Envoy proxy.

_Note: This project is a work in progress. See the [demo instructions](demo/README.md) to get a sense of what we've accomplished and are working on._

## OSM Design

Read more about the high level goals, design and architecture [here](DESIGN.md).

## Managing Services Using OSM

#### On-boarding services to the OSM managed service mesh

To on-board a service to the OSM managed service mesh, OSM first needs to be configured to monitor the namespace the service belongs to. This can be done by labeling the namespace with the mesh name as follows.
```
$ kubectl label namespace <namespace> openservicemesh.io/monitored-by=<mesh-name>
```
The `mesh-name` is a unique identifier assigned to an osm-controller instance during install to identify and manage manage a mesh instance.

After a namespace is labeled for monitoring, new services deployed in a monitored namespace will be a part of the service mesh and OSM will perform automatic sidecar injection for newly created PODs in that namespace.

#### Disabling automatic sidecar injection
Since the sidecar is automatically injected for PODs belonging to a monitored namespace, PODs that are not a part of the service mesh but belong to a monitored namespace should be configured to not have the sidecar injected. This can be achieved by any of the following ways.

- Deploying PODs that are not a part of the service mesh in namespaces that are not monitored by OSM
- Explicitly annotating PODs with sidecar injection as disabled: `"openservicemesh.io/sidecar-injection": "disabled"`

#### Adding existing services to be managed by a new OSM instance
Currently OSM only supports automatic sidecar injection for newly created PODs. Thus, existing services will need to be enabled for monitoring as described above, and then the PODs will need to be redeployed. This workflow will be simplified once OSM supports manual sidecar injection.

#### Un-managing namespaces
To stop OSM from monitoring a namespace, remove the monitoring label from the namespace.
```
$ kubectl label namespace <namespace> openservicemesh.io/monitored-by-
```

[1]: https://en.wikipedia.org/wiki/Service_mesh
[2]: https://github.com/servicemeshinterface/smi-spec/blob/master/SPEC.md
