# Open Service Mesh (OSM)

[![build](https://github.com/open-service-mesh/osm/workflows/Go/badge.svg)](https://github.com/open-service-mesh/osm/actions?query=workflow%3AGo)
[![report](https://goreportcard.com/badge/github.com/open-service-mesh/osm)](https://goreportcard.com/report/github.com/open-service-mesh/osm)
[![codecov](https://codecov.io/gh/open-service-mesh/osm/branch/master/graph/badge.svg)](https://codecov.io/gh/open-service-mesh/osm)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/open-service-mesh/osm/blob/master/LICENSE)
[![release](https://img.shields.io/github/release/open-service-mesh/osm/all.svg)](https://github.com/open-service-mesh/osm/releases)

Open Service Mesh (OSM) is a light weight, Envoy based, [SMI][2] compliant service mesh for applications running in Kubernetes.

## Overview

It is no secret that although microservice environments enable portability, faster and more frequent deployment cycles and even simpler organizational structure via specialized teams, they also increase the complexity of deployments, make it harder to debug existing applications and secure applications in dynamic environments at scale. Using a [service mesh][1] reduces some of the operational burden of microservice environments with a single, dedicated layer of infrastructure for managing service-to-service communication.

The OSM project consists of a control plane which is a set of components installed in a single namespace in a Kubernetes cluster and a command line tool called `osm` for working with the control plane and applications running within the mesh. Once an application is added to the mesh, the OSM control plane installs an Envoy proxy as a sidecar container next to each instance of the application (inside each application Pod in Kubernetes) which then manages all traffic to and from the application. Once the proxy is configured, users have fine grained control on service to service communication and visibility and consistency of metrics for debugging and monitoring without having to touch application code. OSM aims to be simple to install and run while empowering end users with the following features:

1. Secure service to service communication by enabling mTLS and fine grained access control policies.
1. Manage deployments more easily and transparently with various deployment strategies (Canary, A/B testing) for applications running on Kubernetes.
1. Get simple to understand and consistent insights into application metrics for debugging and monitoring with Grafana dashboards.
1. Integrate with external external certificate management services/solutions with a pluggable interface.
1. Onboard applications onto the mesh by enabling automatic sidecar injection of Envoy proxy.

Note: This project is a work in progress. See the [demo instructions](demo/README.md) to get a sense of what we've accomplished and are working on.

## Getting Started

## Prerequisites
- Running Kubernetes cluster at v1.17.0 or later
- A private container registry (temporary requirement as this is currently a private repo)

### Install

The simplest way of installing open service mesh on a Kubernetes cluster is by using the `osm` CLI.

Binary downloads of the OSM client can be found on the [Releases page](https://github.com/open-service-mesh/osm/releases).
Unpack the `osm` binary and add it to your PATH to get started.

See the [installation guide](docs/installation_guide.md) for more options.

### Quickstart

To rapidly get OSM up and running, see the [Quickstart Guide](docs/quickstart_guide.md).

## OSM Design

Read more about the high level goals, design and architecture [here](DESIGN.md).

## Managing Services Using OSM

#### On-boarding services to the OSM managed service mesh

To on-board a service to the OSM managed service mesh, OSM first needs to be configured to monitor the namespace the service belongs to. This can be done by labeling the namespace with the OSM ID as follows.
```
$ kubectl label namespace <namespace> openservicemesh.io/monitor=<osm-id>
```
The `osm-id` is a unique ID for an OSM instance generated during OSM install.

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
$ kubectl label namespace <namespace> openservicemesh.io/monitor-
```

[1]: https://en.wikipedia.org/wiki/Service_mesh
[2]: https://github.com/servicemeshinterface/smi-spec/blob/master/SPEC.md
