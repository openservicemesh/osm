---
title: "OSM Demo: First Steps"
description: "This demo of OSM is a walkthrough of setting up OSM, 2 pods, and applying an SMI policy to allow traffic between them."
type: docs
aliases: ["OSM Simple Demo"]
weight: 2
release: 0.8.0
---


# Open Service Mesh Demo: First Steps


> Note: This demo is specific to OSM v0.8.0


This document will walk you through the steps to:
  - install Open Service Mesh (OSM) v0.8.0
  - install 2 sample apps: `bookbuyer` and `bookstore`
  - apply [Service Mesh Interface (SMI)](https://smi-spec.io/) policies to control the traffic between these pods


## Prerequisites
This demo of OSM v0.8.0 requires:
  - a cluster running Kubernetes v1.18.0 or greater
  - a workstation capable of executing [Bash](https://en.wikipedia.org/wiki/Bash_(Unix_shell)) scripts
  - [The Kubernetes command-line tool](https://kubernetes.io/docs/tasks/tools/#kubectl) - `kubectl`


## Download the OSM command-line tool

The `osm` command-line tool contains everything needed to install and configure Open Service Mesh.
The binary is available on the [OSM GitHub releases page](https://github.com/openservicemesh/osm/releases/).

### For GNU/Linux or macOS
Download and unzip the 64-bit [GNU/Linux](https://github.com/openservicemesh/osm/releases/download/v0.8.0/osm-v0.8.0-linux-amd64.tar.gz)
or
[macOS](https://github.com/openservicemesh/osm/releases/download/v0.8.0/osm-v0.8.0-darwin-amd64.tar.gz)
binary of OSM v0.8.0:
```bash
system=$(uname -s | tr '[:upper:]' '[:lower:]')
release=v0.8.0
curl -L https://github.com/openservicemesh/osm/releases/download/${release}/osm-${release}-${system}-amd64.tar.gz | tar -vxzf -
./${system}-amd64/osm version
```

### For Windows
Download and unzip the Windows OSM v0.8.0 binary via PowerShell:
```powershell
wget  https://github.com/openservicemesh/osm/releases/download/v0.8.0/osm-v0.8.0-windows-amd64.zip -o osm.zip
unzip osm.zip
.\windows-amd64\osm.exe version
```

### Compile from source
The `osm` CLI can be compiled from source using [this guide](../../install/_index.md).


## Installing OSM on Kubernetes

With the `osm` binary downloaded and unzipped, we are ready to install Open Service Mesh on a Kubernetes cluster:

```bash
osm install
```

> Note: This document assumes you have already installed credentials for a Kubernetes cluster in ~/.kube/config and `kubectl cluster-info` executes successfully.

This installed OSM Controller in the `osm-system` namespace.

## Deploy Demo Applications
In this section we will deploy 2 pods in 2 different namespaces:
- `bookbuyer` - an HTTP client, which makes requests to a `bookstore`
- `bookstore` - an HTTP server, which responds to `bookbuyer`


### Create the Namespaces
Each application in this demo will reside in separate namespaces. Create the namespaces with:
```bash
kubectl create namespace bookbuyer
kubectl create namespace bookstore
```

### Join the new namespaces to the service mesh
The following `osm` CLI command joins the namespaces to the service mesh:

`//TODO(draychev): link to relevant documentation on how the JOIN command works [#2904]`

```bash
osm namespace add bookbuyer
osm namespace add bookstore
```

With this command each one of the two namespaces will be:
  1. labelled with `openservicemesh.io/monitored-by: osm`
  2. annotated with `openservicemesh.io/sidecar-injection: enabled`

The OSM Controller will notice the new label and annotation on
these namespaces and will begin injecting **new** pods with Envoy sidecars.

### Create Pods

Create the `bookbuyer` service account:
```bash
kubectl apply -f - <<EOF
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: bookbuyer
  namespace: bookbuyer
EOF
```

Create the `bookbuyer` pod:
```bash
kubectl apply -f - <<EOF
---
apiVersion: v1
kind: Pod
metadata:
  namespace: bookbuyer
  name: bookbuyer
  labels:
    app: bookbuyer
    version: v1
spec:
  serviceAccountName: bookbuyer
  containers:
  - name: bookbuyer
    image: openservicemesh/bookbuyer:v0.8.0
    command: ["/bookbuyer"]
    env:
    - name: "BOOKSTORE_NAMESPACE"
      value: bookstore
EOF
```

Create bookstore service account:
```bash
kubectl apply -f - <<EOF
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: bookstore
  namespace: bookstore
EOF
```

Create bookstore service:
```bash
kubectl apply -f - <<EOF
---
apiVersion: v1
kind: Service
metadata:
  name: bookstore
  namespace: bookstore
  labels:
    app: bookstore
spec:
  selector:
    app: bookstore
  ports:
  - port: 14001
EOF
```

Create bookstore pod:
```bash
kubectl apply -f - <<EOF
---
apiVersion: v1
kind: Pod
metadata:
  namespace: bookstore
  name: bookstore
  labels:
    app: bookstore
spec:
  serviceAccountName: bookstore
  containers:
  - name: bookstore
    image: openservicemesh/bookstore:v0.8.0
    ports:
      - containerPort: 14001
    command: ["/bookstore"]
    args: ["--port", "14001"]
    env:
    - name: BOOKWAREHOUSE_NAMESPACE
      value: bookwarehouse
    - name: IDENTITY
      value: bookstore
EOF
```

View the pods, services, and endponits created so far:

```bash
kubectl get pods -n bookbuyer
kubectl get pods -n bookstore

kubectl get services -n bookbuyer
kubectl get services -n bookstore

kubectl get endpoints -n bookbuyer
kubectl get endpoints -n bookstore
```

## No Access, Yet!

## View the Applications Logs
View the logs of the bookbuyer:
```bash
kubectl logs -n bookbuyer bookbuyer -c bookbuyer
```

View the logs of the bookstore:
```bash
kubectl logs -n bookstore bookstore -c bookstore
```

You'll notice that the HTTP GET requests from `bookbuyer` to `bookstore` are **failing** with an error:
```log
Error fetching http://bookstore.bookstore:14001/buy-a-book/new:
  Get "http://bookstore.bookstore:14001/buy-a-book/new":
    dial tcp 10.0.22.160:14001:
      connect: connection refused
```

We have not applied any traffic policies yet.
[Service Mesh Interface](https://smi-spec.io/) (and OSM by proxy) denies requests where no explicit policy is set.
To permit HTTP GET calls from `bookbuyer` to `bookstore` we need to apply an SMI policy.

## Applying SMI Policies

With SMI policies we are going to:

1. authorize access between pods (using their service accounts)
2. define access control and routing rules for granular traffic management

### Access Control

Apply this [TrafficTarget](https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-access/v1alpha3/traffic-access.md#traffictarget)
policy to allow traffic from bookbuyer to bookstore:
```bash
kubectl apply -f - <<EOF
---
kind: TrafficTarget
apiVersion: access.smi-spec.io/v1alpha3
metadata:
  name: bookstore
  namespace: bookstore
spec:
  destination:
    kind: ServiceAccount
    name: bookstore
    namespace: bookstore
  rules:
  - kind: HTTPRouteGroup
    name: bookstore-service-routes
    matches:
    - all-routes
  sources:
  - kind: ServiceAccount
    name: bookbuyer
    namespace: bookbuyer
EOF
```

Apply this [HTTPRouteGroup](https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-specs/v1alpha4/traffic-specs.md#httproutegroup)
for finer granularity of HTTP traffic control:
```bash
kubectl apply -f - <<EOF
---
apiVersion: specs.smi-spec.io/v1alpha4
kind: HTTPRouteGroup
metadata:
  name: bookstore-service-routes
  namespace: bookstore
spec:
  matches:
  - name: all-routes
    pathRegex: ".*"
    methods:
    - GET
EOF
```

Take another look at the `bookbuyer` logs with:
```bash
kubectl logs -n bookbuyer bookbuyer -c bookbuyer
```

You will see the earlier error is gone and now `bookstore` is responding with `Status: 200 OK`.


# Summary
1. We installed OSM with `osm install`
2. Installed demo apps `bookbuyer` and `bookstore` in 2 separate pods
3. Allowed traffic between the 2 pods with a [TrafficTarget](https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-access/v1alpha3/traffic-access.md#traffictarget) and a [HTTPRouteGroup](https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-specs/v1alpha4/traffic-specs.md#httproutegroup)


# Next Steps
`//TODO(draychev): add a link to the next demo`
