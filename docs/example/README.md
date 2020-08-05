# OSM Example Usage Guide

- [Configure Prerequisites](#configure-prerequisites)
- [Install OSM CLI](#install-osm-cli)
- [Install OSM Control Plane](#install-osm-control-plane)
- [Install Applications](#install-applications)
- [Access Control Policies](#access-control-policies)
- [Traffic Encryption](#traffic-encryption)
- [Traffic Split](#traffic-split-configuration)
- [Inspect Dashboards](#inspect-dashboards)

## Configure Prerequisites
- Have running Kubernetes cluster
- Have `kubectl` CLI installed

## Install OSM CLI
Use the [installation guide](/docs/installation_guide.md) to install the `osm` cli.

## Install OSM Control Plane
```console
$ osm install
```

## Install Applications
The `Bookstore`, `Bookbuyer`, `Bookthief`, `Bookwarehouse` demo applications will be installed in their respective Kubernetes Namespaces. In order for these applications to be injected with a Envoy sidecar automatically, we must add the Namespaces to be monitored by the mesh.

```console
$ kubectl create ns bookstore
$ kubectl create ns bookbuyer
$ kubectl create ns bookthief
$ kubectl create ns bookwarehouse
$ osm namespace add bookstore bookbuyer bookthief bookwarehouse
```

Install `Bookstore`, `Bookbuyer`, `Bookthief`, `Bookwarehouse`.
```console
$ kubectl apply -f docs/example/manifests/apps/
```

### What Got Installed
The manifests include the Kubernetes Deployment, Kubernetes Service, and Kubernetes ServiceAccount for each application.

### View Application UIs
```console
$ ./scripts/port-forward-all.sh
```

In a browser, open up the following urls:
- http://localhost:8080
- http://localhost:8081
- http://localhost:8082
- http://localhost:8083

Postion the windows so that you can see all four at the same time. The header at the top of the webpage indicates the application and version. The window for `bookstore-v2` should not be working. Keep it open anyway for the Traffic Split Configuration section later in this document.

## Access Control Policies
At this point, no applications have access to each other because no access control policies have been applied. Confirm this by confirming that none of the counters in the UI are incrementing. Apply the [SMI Traffic Target][1] and [SMI Traffic Specs][2] resources to define access control policies below:
```console
kubectl apply -f docs/example/manifests/access/
```
The counters should now be incrementing for the windows with headers: `Bookbuyer`, `Bookthief`, and `Bookstore-v1`.

### Going Further
Comment out the lines in the TrafficTarget yaml that allows `Bookthief` to communicate with `Bookstore`. Then, re-apply the manifest and watch the change in policy propogate.
```console
kubectl apply -f example/docs/manifests/access/
```
The counter in the `Bookthief` window should stop incrementing.

*Note: Bypass setting up and using access control policies entirely by enabling permissive traffic policy mode when installing a control plane: `osm install --enable-permissive-traffic-policy`*

## Traffic Encryption
All traffic is encrypted via mTLS regardless of whether you're using access control policies or have enabled premissive traffic policy mode.

## Traffic Split Configuration
Deploy v2 of the `Bookstore` app and use [SMI TrafficSplit][3] to define what percentage of traffic should be applied to each version of `Bookstore`

### Set up Kubernetes Services
Create a Kubernetes Service for each version of `Bookstore`. One Service to send traffic to v1 Pods and another Service to send traffic to v2 Pods.
```console
$ kubectl apply -f docs/example/manifests/bookstore-services/
```

### Apply Traffic Split
Apply a traffic split configuration that sends 100% of the traffic to v1 and 0% to v2. The Kubernetes Service that was originally created with the `Bookstore` v1 deployment will now serve as a *root* or *apex* service. All traffic from any application can be directed to the *root* service and will be managed by this `TrafficSplit` resource.
```console
$ kubectl apply -f docs/example/manifests/split/traffic-split.yaml
```

### Deploy v2 of Bookstore
```
$ kubectl apply -f docs/example/manifests/bookstore-v2/
```
Refresh all browser windows. Observe that the `bookstore-v1` application is now working but the counter in the `bookstore-v2` window is not incrementing.

### Update Traffic Split
Modify the `weight` fields in `traffic-split.yaml` so that each service gets 50% of the traffic and re-apply `traffic-split.yaml`
```console
$ kubectl apply -f docs/example/manifests/split/traffic-split.yaml
```
Wait for changes to propogate and observe that the counter in the `bookstore-v2` window is incrementing.

## Inspect Dashboards
OSM ships with a set of pre-configured Grafana dashboards which can be viewed with the following command:
```console
$ osm dashboard
```

[1]: https://github.com/servicemeshinterface/smi-spec/blob/v0.5.0/apis/traffic-access/v1alpha2/traffic-access.md
[2]: https://github.com/servicemeshinterface/smi-spec/blob/v0.5.0/apis/traffic-specs/v1alpha3/traffic-specs.md
[3]: https://github.com/servicemeshinterface/smi-spec/blob/v0.5.0/apis/traffic-split/v1alpha2/traffic-split.md
