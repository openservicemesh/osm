# Open Service Mesh (OSM)

[![build](https://github.com/openservicemesh/osm/workflows/Go/badge.svg)](https://github.com/openservicemesh/osm/actions?query=workflow%3AGo)
[![report](https://goreportcard.com/badge/github.com/openservicemesh/osm)](https://goreportcard.com/report/github.com/openservicemesh/osm)
[![codecov](https://codecov.io/gh/openservicemesh/osm/branch/main/graph/badge.svg)](https://codecov.io/gh/openservicemesh/osm)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/openservicemesh/osm/blob/main/LICENSE)
[![release](https://img.shields.io/github/release/openservicemesh/osm/all.svg)](https://github.com/openservicemesh/osm/releases)

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

## OSM Design

Read more about the high level goals, design and architecture [here](DESIGN.md).

## Getting Started

Below are quick getting started instructions. For a more detailed example usage guide and demo walkthrough, see [this document](/docs/example/README.md).

### Prerequisites
- Kubernetes cluster running Kubernetes v1.15.0 or greater
- kubectl current context is configured for the target cluster install

### OSM CLI Install

The simplest way of installing Open Service Mesh on a Kubernetes cluster is by using the `osm` CLI.

Download the `osm` binary from the [Releases page](https://github.com/openservicemesh/osm/releases). Unpack the `osm` binary and add it to `$PATH` to get started.
```shell
sudo mv ./osm /usr/local/bin/osm
```

#### Run install pre-flight checks
```shell
$ osm check --pre-install
ok: initialize Kubernetes client
ok: query Kubernetes API
ok: Kubernetes version
ok: can create namespaces
ok: can create customresourcedefinitions
ok: can create clusterroles
ok: can create clusterrolebindings
ok: can create mutatingwebhookconfigurations
ok: can create serviceaccounts
ok: can create services
ok: can create deployments
ok: can create configmaps
ok: can read secrets
ok: can modify iptables
All checks successful!
```

#### Install OSM
```shell
$ osm install
```

See the [installation guide](docs/installation_guide.md) for more detailed options.

### Using OSM

After installing OSM, [onboard Kubernetes services](docs/onboard_services.md) to the service mesh.

### OSM Usage patterns

Refer to [docs/patterns](docs/patterns) for OSM usage patterns.

## Demo and deployment sample

The repository contains tools and scripts to compile and deploy the OSM control plane and an automated demo to show how OSM can manage, secure and provide observability for microservice environments; all on a user-provisioned Kubernetes cluster.
See the [demo instructions](demo/README.md) to get a sense of what we've accomplished and are working on.

For a more manual walkthrough of the same demo, see this [example usage guide](/docs/example/README.md)

## Community

You can reach the Open Service Mesh community and developers via the following channels:

- OSM Slack (TBD)
- Public Community Call (TBD)
- [Mailing list](https://groups.google.com/g/openservicemesh)

## Development Guide

If you would like to contribute to OSM, check out the [development guide](docs/development_guide.md)

## Code of Conduct

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/). See [CODE_OF_CONDUCT.MD](CODE_OF_CONDUCT.MD) for further details.

## License

This software is covered under the MIT license. You can read the license [here](LICENSE).


[1]: https://en.wikipedia.org/wiki/Service_mesh
[2]: https://github.com/servicemeshinterface/smi-spec/blob/master/SPEC_LATEST_STABLE.md
