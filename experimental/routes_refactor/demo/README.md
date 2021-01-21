# README

This document contains instructions for running and experimenting with the Routes V2 feature.

## Prerequisites

### Clone OSM repo

```console
$ git clone git@github.com:openservicemesh/osm.git
```

### Create a Kubernetes cluster

```console
$ make kind-up
```

### Build OSM CLI

```
$ make build-osm
```

### Build and Push OSM control plane assets

```console
$ make docker-push
```

## Install OSM control plane

Install OSM control plane using the `osm` cli with the experimental routes v2 flag enabled
```console
$ bin/osm install --enable-routes-v2-experimental=true --container-registry "$CTR_REGISTRY" --osm-image-tag "CTR_TAG" --osm-chart-path charts/osm
```

## Install Demo manifests

```console
$ bin/osm namespace add bookstore bookbuyer bookthief bookwarehouse
$ cd experimental/routes_refactor_demo/
$ kubectl apply -f manifests/
```

## Verify Demo Successful

In `experimental/routes_refactor/demo/`:
```console
$ ./scripts/port-forward-all.sh
```

- Verify bookbuyer is able to buy books via the counter found at http://localhost:8080
- Verify bookstore-v1 is selling books via the counter found at http://localhost:8081
- Verify bookstore-v1 is _not_ selling books via the counter found at http://localhost:8082
- Verify bookthief is not stealing books via the counter found at http://localhost:8083