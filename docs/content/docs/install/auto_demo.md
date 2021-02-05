---
title: "OSM Automated Demo"
description: "The automated demo is a set of scripts anyone can run and shows how OSM can manage, secure and provide observability for microservice environments."
type: docs
aliases: ["OSM Automated Demo"]
weight: 1
---

# How to Run the OSM Automated Demo

## System Requirements

- MacOS, Linux or WSL2 on Windows
- GCC
- Go version 1.15 or higher
- Kubectl version 1.15 or higher
- Docker CLI
  - on a Debian based GNU/Linux system: `sudo apt-get install docker`
  - on a macOS use `brew install docker` or alternatively visit [Docker for Mac](https://docs.docker.com/docker-for-mac/install/)
  - on Windows visit [Docker for Windows](https://docs.docker.com/docker-for-windows/install/)
- [Watch](http://www.linfo.org/watch.html)
  - `brew install watch` on macOS

## Prerequisites

1. Clone this repo on your workstation
2. Setup `.env` environment variable file
   - From the root of the repository run `make .env`
   - It is already listed in `.gitignore` so that anything you put in it would not accidentally leak into a public git repo. Refer to `.env.example` in the root of this repo for the mandatory and optional environment variables.
3. Provision access to a Kubernetes cluster. Any certified conformant Kubernetes cluster (version 1.15 or higher) can be used. Here are a couple of options:

   - **Option 1:** Local [kind](https://kind.sigs.k8s.io/) cluster
     - [Install kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
       - `brew install kind` on macOS
     - Provision a local cluster and registry in Docker: `make kind-up`
   - **Option 2:** A Kubernetes cluster - use an already provisioned cluster config, either in the default location ($HOME/.kube/config) or referenced by the $KUBECONFIG environment variable.

   We will use images from [Docker Hub](https://hub.docker.com/r/openservicemesh/osm-controller). Ensure you can pull these containers using: `docker pull openservicemesh/osm-controller`

## Run the Demo

From the root of this repository execute:

```shell
./demo/run-osm-demo.sh
```

### Observability

By default, Prometheus is deployed by the demo script. To turn this off. Set the variable `DEPLOY_PROMETHEUS` in your `.env` file to false.
By default, Grafana is deployed by the demo script. To turn this off. Set the variable `DEPLOY_GRAFANA` in your `.env` file to false.

### This script will:

- compile OSM's control plane (`cmd/osm-controller`), create a separate container image and push it to the workstation's default container registry (See `~/.docker/config.json`)
- build and push demo application images described below
- create the following topology in Kubernetes:

  ![Graph](graph.svg)

  - `bookbuyer` and `bookthief` continuously issue HTTP `GET` requests against `bookstore` to buy books and github.com to verify egress traffic.
  - `bookstore` is a service backed by two servers: `bookstore-v1` and `bookstore-v2`. Whenever either sells a book, it issues an HTTP `POST` request to the `bookwarehouse` to restock.

- applies SMI traffic policies allowing `bookbuyer` to access `bookstore-v1` and `bookstore-v2`, while preventing `bookthief` from accessing the `bookstore` services
- installs Jaeger and points all Envoy pods to it
- finally, a command indefinitely watches the relevant pods within the Kubernetes cluster

To see the results of deploying the services and the service mesh - run the tailing scripts:

- the scripts will connect to the respective Kubernetes Pod and stream its logs
- the output will be the output of the curl command to the `bookstore` service and the count of books sold, and the output of the curl command to `github.com` to demonstrate access to an external service
- a properly working service mesh will result in HTTP 200 OK response code for the `bookstore` service with `./demo/tail-bookbuyer.sh` along with a monotonically increasing counter appearing in the response headers, while `./demo/tail-bookthief.sh` will result in HTTP 404 Not Found response code for the `bookstore` service. When egress is enabled, HTTP requests to an out-of-mesh host will result in a HTTP `200 OK` response code for both the `bookbuyer` and `bookthief` services.
  This can be automatically checked with `go run ./ci/cmd/maestro.go`

## View Mesh Topology with Jaeger

The OSM demo will install a Jaeger pod, and configure all participating Envoys to send spans to it. Jaeger's UI is running on port 16686. To view the web UI, forward port 16686 from the Jaeger pod to the local workstation and navigate to http://localhost:16686/. In the `./scripts` directory we have included a helper script to find the Jaeger pod and forward the port: `./scripts/port-forward-jaeger.sh`

## Demo Web UI

The Bookstore, Bookbuyer, and Bookthief apps have simple web UI visualizing the number of requests made between the services.

- To see the UI for Bookbuyer run `./scripts/port-forward-bookbuyer-ui.sh` and open [http://localhost:8080/](http://localhost:8080/)
- To see the UI for Bookstore v1 run `./scripts/port-forward-bookstore-ui-v1.sh` and open [http://localhost:8081/](http://localhost:8081/)
- To see the UI for Bookstore v2 run `./scripts/port-forward-bookstore-ui-v2.sh` and open [http://localhost:8082/](http://localhost:8082/)
- To see the UI for BookThief run `./scripts/port-forward-bookthief-ui.sh` and open [http://localhost:8083/](http://localhost:8083/)
- To see Jaeger run `./scripts/port-forward-jaeger.sh` and open [http://localhost:16686/](http://localhost:16686/)
- To see Grafana run `./scripts/port-forward-grafana.sh` and open [http://localhost:3000/](http://localhost:3000/) - default username and password for Grafana is `admin`/`admin`
- OSM controller has a simple debugging web endpoint - run `./scripts/port-forward-osm-debug.sh` and open [http://localhost:9092/debug](http://localhost:9092/debug)

To expose web UI ports of all components of the service mesh the local workstation use the following helper script: `/scripts/port-forward-all.sh`

## Deleting the kind cluster

When you are done with the demo and want to clean up your local kind cluster, just run the following.

```shell
make kind-reset
```
