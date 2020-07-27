# Open Service Mesh (OSM)

[![build](https://github.com/open-service-mesh/osm/workflows/Go/badge.svg)](https://github.com/open-service-mesh/osm/actions?query=workflow%3AGo)
[![report](https://goreportcard.com/badge/github.com/open-service-mesh/osm)](https://goreportcard.com/report/github.com/open-service-mesh/osm)
[![codecov](https://codecov.io/gh/open-service-mesh/osm/branch/main/graph/badge.svg)](https://codecov.io/gh/open-service-mesh/osm)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/open-service-mesh/osm/blob/main/LICENSE)
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

## Getting Started

### Prerequisites
- Kubernetes cluster running Kubernetes v1.15.0 or greater
- A private container registry (temporary requirement as this is currently a private repo)

### Install

The simplest way of installing open service mesh on a Kubernetes cluster is by using the `osm` CLI.

Download the `osm` binary from the [Releases page](https://github.com/open-service-mesh/osm/releases).

See the [installation guide](docs/installation_guide.md) for more detailed options. After installing OSM, [onboard services](docs/onboard_services.md) to the mesh.

For instructions on how to utilize [Azure Arc](https://azure.microsoft.com/en-us/services/azure-arc/) and the [Azure Arc-enabled Kubernetes](https://docs.microsoft.com/en-us/azure/azure-arc/kubernetes/overview) experience for deploying OSM and configuring policies, please visit the [OSM Azure Arc Installation Guide](docs/azure_arc_installation_guide.md).

## Community, discussion, contribution, and support

You can reach the Open Service Mesh community and developers via the following channels:

- OSM Slack (TBD)
- Public Community Call (TBD)

## Code of Conduct

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/). See [CODE_OF_CONDUCT.MD](CODE_OF_CONDUCT.MD) for further details.

## License

This software is covered under the MIT license. You can read the license [here](LICENSE).


[1]: https://en.wikipedia.org/wiki/Service_mesh
[2]: https://github.com/servicemeshinterface/smi-spec/blob/master/SPEC_LATEST_STABLE.md
