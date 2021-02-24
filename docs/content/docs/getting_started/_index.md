---
title: "Getting Started"
description: "Open Service Mesh (OSM) is a lightweight, extensible, Cloud Native service mesh that allows users to uniformly manage, secure, and get out-of-the-box observability features for highly dynamic microservice environments."
type: docs
aliases: ["getting started"]
weight: 1
---

The OSM project builds on the ideas and implementations of many cloud native ecosystem projects including [Linkerd](https://github.com/linkerd/linkerd), [Istio](https://github.com/istio/istio), [Consul](https://github.com/hashicorp/consul), [Envoy](https://github.com/envoyproxy/envoy), [Kuma](https://github.com/kumahq/kuma), [Helm](https://github.com/helm/helm), and the [SMI](https://github.com/servicemeshinterface/smi-spec) specification.

## Overview

OSM runs an Envoy based control plane on Kubernetes, can be configured with SMI APIs, and works by injecting an Envoy proxy as a sidecar container next to each instance of your application. The proxy contains and executes rules around access control policies, implements routing configuration, and captures metrics. The control plane continually configures proxies to ensure policies and routing rules are up to date and ensures proxies are healthy.

## Core Principles

1. Simple to understand and contribute to
1. Effortless to install, maintain, and operate
1. Painless to troubleshoot
1. Easy to configure via [Service Mesh Interface (SMI)](https://github.com/servicemeshinterface/smi-spec/blob/master/SPEC_LATEST_STABLE.md)

## Features

1. Easily and transparently configure [traffic shifting](https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-split/v1alpha2/traffic-split.md) for deployments
1. Secure service to service communication by [enabling mTLS](https://github.com/openservicemesh/osm/blob/main/docs/patterns/certificates.md)
1. Define and execute fine grained [access control](https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-access/v1alpha3/traffic-access.md) policies for services
1. [Observability](https://github.com/openservicemesh/osm/blob/main/docs/patterns/observability/README.md) and insights into application metrics for debugging and monitoring services
1. Integrate with [external certificate management](https://github.com/openservicemesh/osm/blob/main/docs/patterns/certificates.md) services/solutions with a pluggable interface
1. Onboard applications onto the mesh by enabling [automatic sidecar injection](https://github.com/openservicemesh/osm/blob/main/docs/patterns/sidecar_injection.md) of Envoy proxy

## Project Status

OSM is under active development and is NOT ready for production workloads.

## Support

OSM is an open source project that is [not covered by the Microsoft Azure support policy](https://support.microsoft.com/en-us/help/2941892/support-for-linux-and-open-source-technology-in-azure). [Please search open issues here](https://github.com/openservicemesh/osm/issues), and if your issue isn't already represented please open a new one. The OSM project maintainers will respond to the best of their abilities.

## SMI Specification Support

| Specification Component |                                                     Supported Release                                                     |                                    Comments                                     |
| :---------------------- | :-----------------------------------------------------------------------------------------------------------------------: | :-----------------------------------------------------------------------------: |
| Traffic Access Control  |  [v1alpha3](https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-access/v1alpha3/traffic-access.md)  |                                                                                 |
| Traffic Specs           |   [v1alpha4](https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-specs/v1alpha4/traffic-specs.md)   |                                                                                 |
| Traffic Split           |   [v1alpha2](https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-split/v1alpha2/traffic-split.md)   |                                                                                 |
| Traffic Metrics         | [v1alpha1](https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-metrics/v1alpha1/traffic-metrics.md) | 🚧 **In Progress** [#379](https://github.com/openservicemesh/osm/issues/379) 🚧 |
