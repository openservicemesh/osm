# OSM Manual Demo Guide

## Table of Contents
- [Overview](#overview)
- [Configure Prerequisites](#configure-prerequisites)
- [Install OSM CLI](#install-osm-cli)
- [Install OSM Control Plane](#install-osm-control-plane)
- [Deploying the Bookstore Demo Applications](#deploying-the-bookstore-demo-applications)
  - [Create the Bookstore Applications Namespaces](#create-the-bookstore-application-namespaces)
  - [Onboard the Namespaces to the OSM Mesh and enable sidecar injection](#onboard-the-namespaces-to-the-osm-mesh-and-enable-sidecar-injection-on-the-namespaces)
  - [Deploy the Bookstore Application](#deploy-the-bookstore-application)
  - [Checkpoint: What got installed?](#checkpoint-what-got-installed)
  - [View the Applications UIs](#view-the-application-uis)
- [Deploy SMI Access Control Policies](#deploy-smi-access-control-policies)
  - [Allowing the Bookthief Application to access the Mesh](#allowing-the-bookthief-application-to-access-the-mesh)
- [Traffic Encryption](#traffic-encryption)
- [Configure Traffic Split between Services](#configure-traffic-split-between-two-services)
  - [Deploy v2 of Bookstore](#deploy-v2-of-bookstore)
  - [Update Traffic Split](#update-traffic-split)
- [Inspect Dashbaords](#inspect-dashboards)

## Overview
The OSM Manual Install Demo Guide is designed to quickly allow you to demo and experience the OSM mesh.

## Configure Prerequisites
- Kubernetes cluster running Kubernetes v1.15.0 or greater
- Have `kubectl` CLI installed - [Install and Set Up Kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- kubectl current context is configured for the target cluster install
  - ```kubectl config current-context```
- Have a local clone of the OSM GitHub Repo
  - ```git clone https://github.com/openservicemesh/osm.git```
  - ```cd osm```



## Install OSM CLI
Use the [installation guide](/docs/installation_guide.md) to install the `osm` cli.

## Install OSM Control Plane

1.  By default, OSM does not enable Prometheus or Grafana.
    ```bash
    osm install
    ```
1. To enable Prometheus and Grafana, use their respective flags
    ```bash
    osm install --deploy-prometheus true --deploy-grafana true
    ```
    See the [metrics documentation](/docs/patterns/observability/metrics.md#automatic-bring-up) for more details.

## Deploying the Bookstore Demo Applications
The `Bookstore`, `Bookbuyer`, `Bookthief`, `Bookwarehouse` demo applications will be installed in their respective Kubernetes Namespaces. In order for these applications to be injected with a Envoy sidecar automatically, we must add the Namespaces to be monitored by the mesh.

### Create the Bookstore Application Namespaces
```bash
for i in bookstore bookbuyer bookthief bookwarehouse; do kubectl create ns $i; done
```
### Onboard the Namespaces to the OSM Mesh and enable sidecar injection on the namespaces
```bash
osm namespace add bookstore bookbuyer bookthief bookwarehouse
```
### Deploy the Bookstore Application
Install `Bookstore`, `Bookbuyer`, `Bookthief`, `Bookwarehouse`.
```bash
kubectl apply -f docs/example/manifests/apps/
```

### Checkpoint: What Got Installed?
The following are the key components of the demo application:

- A Kubernetes Deployment, Kubernetes Service, and Kubernetes ServiceAccount for each application.
- A *root service* called `bookstore` which other applications will use to direct traffic to the Bookbuyer application.
- An [SMI TrafficSplit][3] resource which specifies how much traffic should go to each version of `Bookstore`.

To view these resources on your cluster, run the following commands:
```
kubectl get svc --all-namespaces
kubectl get deploy --all-namespaces
kubectl get trafficsplit -n bookstore
```

A simple topology view of the Bookstore application looks like the following:
![Bookstore Application Topology](/img/book-thief-app-topology.jpg "Bookstore Application Topology")

*Note: At the moment, you must configure a TrafficSplit object to get your applications set up correctly for inbound traffic because it helps us properly configure the dataplane. We're working on removing the need for this entirely.* [#1370](https://github.com/openservicemesh/osm/issues/1370)

### View the Application UIs
We will now setup client port forwarding, so we can access the services in the Kubernetes cluster. It is best to start a new terminal session for running the port forwarding script to maintain the port forwarding session, while using the original terminal to continue to issue commands. The port-forward-all.sh script will look for a ```".env"``` file for variables. The ```".env"``` creates the necessary variables that target the previously created namespaces. We will use the reference .env.examples file and then run the port forwarding script.

In a new terminal session, run the following commands to enable port forwarding into the Kubernetes cluster.
```bash
cp .env.example .env
./scripts/port-forward-all.sh
```
*Note: To override the default ports, prefix the `BOOKBUYER_LOCAL_PORT`, `BOOKSTOREv1_LOCAL_PORT`, `BOOKSTOREv2_LOCAL_PORT`, and/or `BOOKTHIEF_LOCAL_PORT` variable assignments to the `port-forward` scripts. For example:*

```bash
BOOKBUYER_LOCAL_PORT=7070 BOOKSTOREv1_LOCAL_PORT=7071 BOOKSTOREv2_LOCAL_PORT=7072 BOOKTHIEF_LOCAL_PORT=7073 ./scripts/port-forward-all.sh
```
In a browser, open up the following urls:
- http://localhost:8080 - **Bookbuyer**
- http://localhost:8081 - **bookstore-v1**
- http://localhost:8082 - **bookstore-v2**
  - *Note: This page will not be available at this time in the demo. This will become available during the Traffic Split Configuration*
- http://localhost:8083 - **bookthief**

Position the windows so that you can see all four at the same time. The header at the top of the webpage indicates the application and version.

## Deploy SMI Access Control Policies
At this point, no applications have access to each other because no access control policies have been applied. Confirm this by confirming that none of the counters in the UI are incrementing. Apply the [SMI Traffic Target][1] and [SMI Traffic Specs][2] resources to define access control policies below:
```bash
kubectl create -f docs/example/manifests/access/
```
The counters should now be incrementing for the `Bookbuyer`, and `Bookstore-v1` applications:
- http://localhost:8080 - **Bookbuyer**
- http://localhost:8081 - **bookstore-v1**

### Allowing the Bookthief Application to access the Mesh
Currently the Bookthief application has not been authorized to participate in the service mesh communication. We will now uncomment out the lines in the [docs/example/manifests/access/traffic-access.yaml](manifests/access/traffic-access.yaml) to allow `Bookthief` to communicate with `Bookstore`. Then, re-apply the manifest and watch the change in policy propagate.

Current TrafficTarget spec with commented `Bookthief` kind:
```
kind: TrafficTarget
apiVersion: access.smi-spec.io/v1alpha3
metadata:
  name: bookstore-v1
  namespace: bookstore
spec:
  destination:
    kind: ServiceAccount
    name: bookstore-v1
    namespace: bookstore
  rules:
  - kind: HTTPRouteGroup
    name: bookstore-service-routes
    matches:
    - buy-a-book
    - books-bought
  sources:
  - kind: ServiceAccount
    name: bookbuyer
    namespace: bookbuyer
  #- kind: ServiceAccount
    #name: bookthief
    #namespace: bookthief
 ```

 Updated TrafficTarget spec with uncommented `Bookthief` kind:
 ```
 kind: TrafficTarget
apiVersion: access.smi-spec.io/v1alpha3
metadata:
  name: bookstore-v1
  namespace: bookstore
spec:
  destination:
    kind: ServiceAccount
    name: bookstore-v1
    namespace: bookstore
  rules:
  - kind: HTTPRouteGroup
    name: bookstore-service-routes
    matches:
    - buy-a-book
    - books-bought
  sources:
  - kind: ServiceAccount
    name: bookbuyer
    namespace: bookbuyer
  - kind: ServiceAccount
    name: bookthief
    namespace: bookthief
 ```

Re-apply the access manifest with the updates.
```bash
kubectl apply -f docs/example/manifests/access/
```
The counter in the `Bookthief` window will start incrementing.
- http://localhost:8083 - **bookthief**

*Note: Bypass setting up and using access control policies entirely by enabling permissive traffic policy mode when installing a control plane: `osm install --enable-permissive-traffic-policy`*

## Traffic Encryption

All traffic is encrypted via mTLS regardless of whether you're using access control policies or have enabled permissive traffic policy mode.

## Configure Traffic Split between Two Services

We will now demonstrate how to balance traffic between two Kubernetes services, commonly known as a traffic split. We will be splitting the traffic between the bookstore-v1 service and the bookstore-v2 service. The bookstore-v1 service is currently deployed. We will deploy the bookstore-v2 service of the `Bookstore` app. Once the bookstore-v2 service is deployed we will then apply the updated `TrafficSplit` configuration between the two services.

### Deploy v2 of Bookstore

A Kubernetes Service, ServiceAccount, and Deployment and SMI TrafficTarget for bookstore-v2 will be applied with the following command:
```bash
kubectl apply -f docs/example/manifests/bookstore-v2/
```

Browse to http://localhost:8082. You should see the `bookstore-v2` heading in your browser window. **NOTE** Please exit and restart the `./scripts/port-forward-all.sh` script in order to access v2 of Bookstore.

After restarting the port forwarding script, you should now be able to access the `bookstore-v2` application at http://localhost:8082. The count for the books sold should remain at 0, this is because the current traffic split policy is currently weighted 100% for `bookstore-v1`. You can verify the traffic split policy by running the following and viewing the **Backends** properties:
```
kubectl describe trafficsplit bookstore-split -n bookstore
```

### Update Traffic Split

An updated SMI TrafficSplit policy for `bookstore` Service configuring all traffic to go to bookstore-v2 will be applied using the following command:
```bash
kubectl apply -f docs/example/manifests/split-v2/
```

Wait for the changes to propagate and observe the counters increment for bookstore-v2 in your browser windows. Modify the `weight` fields in [manifests/split-v2/traffic-split-v2.yaml](manifests/split-v2/traffic-split-v2.yaml) and re-apply changes to experiment.

- http://localhost:8082 - **bookstore-v2**

## Inspect Dashboards
OSM can be configured to deploy Grafana dashboards using the `--deploy-grafana` flag in `osm install`. **NOTE** If you still have the additional terminal still running the `./scripts/port-forward-all.sh` script, go ahead and `CTRL+C` to terminate the port forwarding. The `osm dashboard` port redirection will not work simultaneously with the port forwarding script still running. The `osm dashboard` can be viewed with the following command:
```bash
$ osm dashboard
```
Simply navigate to http://localhost:3000 to access the Grafana dashboards. The default user name is `admin` and the default password is `admin`. On the Grafana homepage click on the **Home** icon, you will see a folders containing dashboards for both OSM Control Plan and OSM Data Plane.

[1]: https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-access/v1alpha2/traffic-access.md
[2]: https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-specs/v1alpha4/traffic-specs.md
[3]: https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-split/v1alpha2/traffic-split.md
