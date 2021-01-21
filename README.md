# Open Service Mesh (OSM)

[![build](https://github.com/openservicemesh/osm/workflows/Go/badge.svg)](https://github.com/openservicemesh/osm/actions?query=workflow%3AGo)
[![report](https://goreportcard.com/badge/github.com/openservicemesh/osm)](https://goreportcard.com/report/github.com/openservicemesh/osm)
[![codecov](https://codecov.io/gh/openservicemesh/osm/branch/main/graph/badge.svg)](https://codecov.io/gh/openservicemesh/osm)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/openservicemesh/osm/blob/main/LICENSE)
[![release](https://img.shields.io/github/release/openservicemesh/osm/all.svg)](https://github.com/openservicemesh/osm/releases)

Open Service Mesh (OSM) is a lightweight, extensible, Cloud Native [service mesh][1] that allows users to uniformly manage, secure, and get out-of-the-box observability features for highly dynamic microservice environments.

The OSM project builds on the ideas and implementations of many cloud native ecosystem projects including [Linkerd](https://github.com/linkerd/linkerd), [Istio](https://github.com/istio/istio), [Consul](https://github.com/hashicorp/consul), [Envoy](https://github.com/envoyproxy/envoy), [Kuma](https://github.com/kumahq/kuma), [Helm](https://github.com/helm/helm), and the [SMI](https://github.com/servicemeshinterface/smi-spec) specification.

## Table of Contents
- [Overview](#overview)
  - [Core Principles](#core-principles)
  - [Features](#features)
  - [Project Status](#project-status)
  - [Support](#support)
  - [SMI Specification Support](#smi-specification-support)
- [OSM Design](#osm-design)
- [Install](#install)
    - [Prerequisites](#prerequisites)
    - [Get the OSM CLI](#get-the-osm-cli)
    - [Install OSM](#install-osm)
- [Demos](#demos)
- [Using OSM](#using-osm)
    - [OSM Usage Patterns](#osm-usage-patterns)
- [Community](#community)
- [Development Guide](#development-guide)
- [Code of Conduct](#code-of-conduct)
- [License](#license)


## Overview

OSM runs an Envoy based control plane on Kubernetes, can be configured with SMI APIs, and works by injecting an Envoy proxy as a sidecar container next to each instance of your application. The proxy contains and executes rules around access control policies, implements routing configuration, and captures metrics. The control plane continually configures proxies to ensure policies and routing rules are up to date and ensures proxies are healthy.

### Core Principles
1. Simple to understand and contribute to
1. Effortless to install, maintain, and operate
1. Painless to troubleshoot
1. Easy to configure via [Service Mesh Interface (SMI)][2]

### Features

1. Easily and transparently configure [traffic shifting][3] for deployments
1. Secure service to service communication by [enabling mTLS](docs/patterns/certificates.md)
1. Define and execute fine grained [access control][4] policies for services
1. [Observability](docs/patterns/observability/README.md) and insights into application metrics for debugging and monitoring services
1. Integrate with [external certificate management](docs/patterns/certificates.md) services/solutions with a pluggable interface
1. Onboard applications onto the mesh by enabling [automatic sidecar injection](docs/patterns/sidecar_injection.md) of Envoy proxy

### Project status

OSM is under active development and is **NOT** ready for production workloads.

### Support

OSM is an open source project that is [**not** covered by the Microsoft Azure support policy](https://support.microsoft.com/en-us/help/2941892/support-for-linux-and-open-source-technology-in-azure). [Please search open issues here](https://github.com/openservicemesh/osm/issues), and if your issue isn't already represented please [open a new one](https://github.com/openservicemesh/osm/issues/new/choose). The OSM project maintainers will respond to the best of their abilities.

### SMI Specification support

|   Specification Component    |         Supported Release          |          Comments          |
| :---------------------------- | :--------------------------------: |  :--------------------------------: |
| Traffic Access Control  |  [v1alpha3](https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-access/v1alpha3/traffic-access.md)  | |
| Traffic Specs  |  [v1alpha4](https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-specs/v1alpha4/traffic-specs.md)  | |
| Traffic Split  |  [v1alpha2](https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-split/v1alpha2/traffic-split.md) | |
| Traffic Metrics  | [v1alpha1](https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-metrics/v1alpha1/traffic-metrics.md) | 🚧 **In Progress** [#379](https://github.com/openservicemesh/osm/issues/379) 🚧 |

## OSM Design

Read more about [OSM's high level goals, design, and architecture](DESIGN.md).

## Install

### Prerequisites
- Kubernetes cluster running Kubernetes v1.15.0 or greater
- kubectl current context is configured for the target cluster install
  - ```kubectl config current-context```

### Get the OSM CLI

The simplest way of installing Open Service Mesh on a Kubernetes cluster is by using the `osm` CLI.

Download the `osm` binary from the [Releases page](https://github.com/openservicemesh/osm/releases). Unpack the `osm` binary and add it to `$PATH` to get started.
```shell
sudo mv ./osm /usr/local/bin/osm
```

### Install OSM
```shell
$ osm install
```
![OSM Install Demo](img/osm-install-demo-v0.2.0.gif "OSM Install Demo")

See the [installation guide](docs/installation_guide.md) for more detailed options.

## Demos
We have provided two demos for you to experience OSM.

- The [automated demo](demo/README.md) is a set of scripts anyone can run and shows how OSM can manage, secure and provide observability for microservice environments.
- The [manual demo](docs/example/README.md) is a step-by-step walkthrough set of instruction of the automated demo.

## Using OSM

After installing OSM, [onboard a microservice application](docs/onboard_services.md) to the service mesh.

### OSM Usage Patterns

1. [Ingress](docs/patterns/ingress.md) and [Egress](docs/patterns/egress.md)
1. [Observability](docs/patterns/observability/README.md)
1. [Certificates](docs/patterns/certificates.md)
1. [Sidecar Injection](docs/patterns/sidecar_injection.md)

## Community

Connect with the Open Service Mesh community:

- GitHub [issues](https://github.com/openservicemesh/osm/issues) and [pull requests](https://github.com/openservicemesh/osm/pulls) in this repo
- OSM Slack: <a href="https://slack.cncf.io/">Join</a> the CNCF Slack for related discussions in <a href="https://cloud-native.slack.com/archives/C018794NV1C">#openservicemesh</a>
- Public Community Call: OSM Community calls take place on the [second Tuesday of each month, 10:30am-11am Pacific](https://calendar.google.com/calendar?cid=Y181dXJwY3F0NWd2OW5ldXE2c2IxM2hvcnN2Z0Bncm91cC5jYWxlbmRhci5nb29nbGUuY29t) in the [CNCF OSM Zoom room](https://zoom.us/my/cncfosm?pwd=aXdkaGU3OWRjUllyaHZEZkh0ZjFwUT09) - notes available in [Open Service Mesh (OSM) Community Meeting Notes](https://docs.google.com/document/d/1da-XIqthmyG7zQyFAV1Kt-Qvq4NoNNBX7hZ_sM_kM98/edit?usp=sharing)
- [Mailing list](https://groups.google.com/g/openservicemesh)

## Development Guide

If you would like to contribute to OSM, check out the [development guide](docs/development_guide.md).

## Code of Conduct

This project has adopted the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md). See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) for further details.

## License

This software is covered under the MIT license. You can read the license [here](LICENSE).


[1]: https://en.wikipedia.org/wiki/Service_mesh
[2]: https://github.com/servicemeshinterface/smi-spec/blob/master/SPEC_LATEST_STABLE.md
[3]: https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-split/v1alpha2/traffic-split.md
[4]: https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-access/v1alpha3/traffic-access.md
