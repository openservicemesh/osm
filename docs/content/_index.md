---
title: "Docs"
description: "Open Service Mesh documentation and resources."
type: docs
---

## Overview

OSM runs an Envoy based control plane on Kubernetes, can be configured with SMI APIs, and works by injecting an Envoy proxy as a sidecar container next to each instance of your application. The proxy contains and executes rules around access control policies, implements routing configuration, and captures metrics. The control plane continually configures proxies to ensure policies and routing rules are up to date and ensures proxies are healthy.

## Core Principles
* Simple to understand and contribute to
* Effortless to install, maintain, and operate
* Painless to troubleshoot
* Easy to configure via Service Mesh Interface (SMI)

## Features
* Easily and transparently configure traffic shifting for deployments
* Secure service to service communication by enabling mTLS
* Define and execute fine grained access control policies for services
* Observability and insights into application metrics for debugging and monitoring services
* Integrate with external certificate management services/solutions with a pluggable interface
* Onboard applications onto the mesh by enabling automatic sidecar injection of Envoy proxy

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

![OSM Install Demo](https://github.com/openservicemesh/osm/raw/main/img/osm-install-demo-v0.2.0.gif "OSM Install Demo")
