# OSM Example Usage Guide

## Table of Contents
- [Configure Prerequisites](#configure-prerequisites)
- [Install OSM CLI](#install-osm-cli)
- [Install OSM Control Plane](#install-osm-control-plane)
- [Install Applications](#install-applications)
  - [What Got Installed](#what-got-installed)
  - [View Application UIs](#view-application-uis)
- [Access Control Policies](#access-control-policies)
  - [Going Further](#going-further)
- [Traffic Encryption](#traffic-encryption)
- [Traffic Split](#traffic-split-configuration)
  - [Deploy v2 of Bookstore](#deploy-v2-of-bookstore)
  - [Update Traffic Split](#update-traffic-split)
- [Inspect Dashboards](#inspect-dashboards)

## Configure Prerequisites
- Have running Kubernetes cluster
- Have `kubectl` CLI installed
- GNU Parallel

## Install OSM CLI
Use the [installation guide](/docs/installation_guide.md) to install the `osm` cli.

## Install OSM Control Plane
```bash
osm install
```

## Deploying the Bookstore Applications
The `Bookstore`, `Bookbuyer`, `Bookthief`, `Bookwarehouse` demo applications will be installed in their respective Kubernetes Namespaces. In order for these applications to be injected with a Envoy sidecar automatically, we must add the Namespaces to be monitored by the mesh.

### 1. Create the Bookstore Application Namespaces
```bash
for i in bookstore bookbuyer bookthief bookwarehouse; do kubectl create ns $i; done
```
### 2. Onboard the Namespaces to the OSM Mesh
```bash
for i in bookstore bookbuyer bookthief bookwarehouse; do osm namespace add $i; done
```
### 3. Deploy the Bookstore Application
Install `Bookstore`, `Bookbuyer`, `Bookthief`, `Bookwarehouse`.
```bash
kubectl create -f docs/example/manifests/apps/
```

### What Got Installed
- A Kubernetes Deployment, Kubernetes Service, and Kubernetes ServiceAccount for each application.
- A *root service* called `bookstore` which other applications will use to direct traffic to the Bookbuyer application.
- An [SMI TrafficSplit][3] resource which specifies how much traffic should to each version of `Bookstore`.

*Note: At the moment, you must configure a TrafficSplit object to get your applications set up correctly for inbound traffic because it helps us properly configure the dataplane. We're working on removing the need for this entirely.* [#1370](https://github.com/openservicemesh/osm/issues/1370)

### View Application UIs
```bash
./scripts/port-forward-all.sh
```

In a browser, open up the following urls:
- http://localhost:8080
- http://localhost:8081
- http://localhost:8082
- http://localhost:8083

Position the windows so that you can see all four at the same time. The header at the top of the webpage indicates the application and version. The window for `bookstore-v2` should not be working. Keep it open anyway for the Traffic Split Configuration section later in this document.

## Access Control Policies
At this point, no applications have access to each other because no access control policies have been applied. Confirm this by confirming that none of the counters in the UI are incrementing. Apply the [SMI Traffic Target][1] and [SMI Traffic Specs][2] resources to define access control policies below:
```bash
kubectl apply -f docs/example/manifests/access/
```
The counters should now be incrementing for the windows with headers: `Bookbuyer`, and `Bookstore-v1`.

### Going Further
Uncomment out the lines in the [manifests/access/traffic-access.yaml](manifests/access/traffic-access.yaml) to allow `Bookthief` to communicate with `Bookstore`. Then, re-apply the manifest and watch the change in policy propagate.
```bash
kubectl apply -f docs/example/manifests/access/
```
The counter in the `Bookthief` window will start incrementing.

*Note: Bypass setting up and using access control policies entirely by enabling permissive traffic policy mode when installing a control plane: `osm install --enable-permissive-traffic-policy`*

## Traffic Encryption

All traffic is encrypted via mTLS regardless of whether you're using access control policies or have enabled permissive traffic policy mode.

## Traffic Split Configuration

Deploy v2 of the `Bookstore` app. Then, apply the updated `TrafficSplit` configuration.

### Deploy v2 of Bookstore

A Kubernetes Service, ServiceAccount, and Deployment and SMI TrafficTarget for bookstore-v2 will be applied with the following command:
```bash
kubectl apply -f docs/example/manifests/bookstore-v2/
```

Browse to http://localhost:8082. You should see the `bookstore-v2` heading in your browser window. **NOTE** Please exit and restart the `./scripts/port-forward-all.sh` script in order to access v2 of Bookstore.

### Update Traffic Split

An updated SMI TrafficSplit policy for `bookstore` Service configuring all traffic to go to bookstore-v2 will be applied using the following command:
```bash
kubectl apply -f docs/example/manifests/split-v2/
```

Wait for the changes to propagate and observe the counters increment for bookstore-v2 in your browser windows. Modify the `weight` fields in [manifests/split-v2/traffic-split-v2.yaml](manifests/split-v2/traffic-split-v2.yaml) and re-apply changes to experiment.

## Inspect Dashboards
OSM ships with a set of pre-configured Grafana dashboards which can be viewed with the following command:
```bash
$ osm dashboard
```
**NOTE** If the `./scripts/port-forward-all.sh` script is still running the `osm dashboard` command will return an error and you can simply navigate to http://localhost:3000 to access the Grafana dashboards.

[1]: https://github.com/servicemeshinterface/smi-spec/blob/v0.5.0/apis/traffic-access/v1alpha2/traffic-access.md
[2]: https://github.com/servicemeshinterface/smi-spec/blob/v0.5.0/apis/traffic-specs/v1alpha3/traffic-specs.md
[3]: https://github.com/servicemeshinterface/smi-spec/blob/v0.5.0/apis/traffic-split/v1alpha2/traffic-split.md
