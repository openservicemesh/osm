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
- [Traffic Policy Modes](#traffic-policy-modes)
- [SMI Traffic Policy Mode](#smi-traffic-policy-mode)
  - [Deploy SMI Access Control Policies](#deploy-smi-access-control-policies)
    - [Allowing the Bookthief Application to access the Mesh](#allowing-the-bookthief-application-to-access-the-mesh)
  - [Configure Traffic Split between Services](#configure-traffic-split-between-two-services)
    - [Split Traffic to v2 of Bookstore](#split-traffic-to-v2-of-bookstore)
    - [Update Traffic Split](#update-traffic-split)
- [Permissive Traffic Policy Mode](#permissive-traffic-policy-mode)
  - [Deploy a service with permissive traffic policy mode enabled](#deploy-a-service-with-permissive-traffic-policy-mode-enabled)
- [Traffic Encryption](#traffic-encryption)
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
Deploy `Bookstore v1`, `Bookstore v2`, `Bookbuyer`, `Bookthief`, `Bookwarehouse` applications:
```bash
kubectl apply -f docs/example/manifests/apps/bookstore-v1.yaml
kubectl apply -f docs/example/manifests/apps/bookstore-v2.yaml
kubectl apply -f docs/example/manifests/apps/bookbuyer.yaml
kubectl apply -f docs/example/manifests/apps/bookthief.yaml
kubectl apply -f docs/example/manifests/apps/bookwarehouse.yaml
```

### Checkpoint: What Got Installed?
A Kubernetes Service, Deployment, and ServiceAccount for applications `bookstore-v1`, `bookstore-v2`, `bookbuyer`, `bookthief`, and `bookwarehouse`.

To view these resources on your cluster, run the following commands:
```
kubectl get svc --all-namespaces
kubectl get deployment --all-namespaces
```

A simple topology view of the Bookstore application looks like the following:
![Bookstore Application Topology](/img/book-thief-app-topology.jpg "Bookstore Application Topology")

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
- http://localhost:8080 - **bookbuyer**
- http://localhost:8081 - **bookstore-v1**
- http://localhost:8082 - **bookstore-v2**
  - *Note: This page will not be available at this time in the demo. This will become available during the Traffic Split Configuration*
- http://localhost:8083 - **bookthief**

Position the windows so that you can see all four at the same time. The header at the top of the webpage indicates the application and version.

## Traffic Policy Modes
Once the applications are up and running, they can interact with each other using [SMI traffic policy mode](#smi-traffic-policy-mode) or [permissive traffic policy mode](#permissive-traffic-policy-mode). In the SMI policy mode, all traffic is denied by default unless explicitly allowed using a combination of SMI access and routing policies. In permissive traffic policy mode, traffic between application services is automatically configured by `osm-controller`, and SMI policies are not enforced.

### Verify the Traffic Policy Mode
Check whether permissive traffic policy mode is enabled or not by retrieving the value for the `permissive_traffic_policy_mode` key in the `osm-config` ConfigMap.
```bash
# Replace osm-system with osm-controller's namespace if using a non default namespace
kubectl get configmap -n osm-system osm-config -o json | jq -r '.data["permissive_traffic_policy_mode"]'
# Output:
# false: permissive traffic policy mode is disabled, SMI policy mode is enabled
# true: permissive traffic policy mode is enabled, SMI policy mode is disabled
```

The following sections demonstrate using OSM with [SMI Traffic Policy Mode](#smi-traffic-policy-mode) and [permissive traffic policy mode](#permissive-traffic-policy-mode).

## SMI Traffic Policy Mode
SMI traffic policies can be used for the following:
1. SMI access control policies to authorize traffic access between service identities
1. SMI traffic specs policies to define routing rules to associate with access control policies
1. SMI traffic split policies to direct client traffic to multiple backends based on weights

The following sections describe how to leverage each of these policies to enforce fine grained control over traffic flowing within the service mesh. Before proceeding, [verify the traffic policy mode](#verify-the-traffic-policy-mode) and ensure the `permissive_traffic_policy_mode` key is set to `false` in the `osm-config` ConfigMap.

SMI traffic policy mode can be enabled by disabling permissive traffic policy mode:
```bash
# Replace osm-system with osm-controller's namespace if using a non default namespace
kubectl patch ConfigMap osm-config -n osm-system -p '{"data":{"permissive_traffic_policy_mode":"false"}}' --type=merge
```

### Deploy SMI Access Control Policies
At this point, no applications have access to each other because no access control policies have been applied. Confirm this by verifying that none of the counters in the UI are incrementing.

Apply the [SMI Traffic Target][1], [SMI Traffic Specs][2], and [SMI TrafficSplit][3] resources to define access control and routing policies for the applications to communicate:
```bash
# Deploy SMI TrafficTarget and HTTPRouteGroup policy
kubectl apply -f docs/example/manifests/access/traffic-access-v1.yaml
```

*Note: At the moment, you must configure a TrafficSplit resource to get your applications set up correctly for inbound traffic because it helps us properly configure the dataplane. We are working on removing the need for this entirely.* [#1370](https://github.com/openservicemesh/osm/issues/1370)

Deploy the `bookstore` root service for the purpose of splitting traffic. Client application `bookbuyer` and `bookthief` will direct traffic to the `bookstore` root service which is referenced in the SMI traffic split policy we will deploy next.
```bash
kubectl apply -f docs/example/manifests/apps/bookstore-root-service.yaml
```

Deploy the SMI traffic split policy to direct 100 percent of the traffic sent to the root `bookstore` service to the `bookstore-v1` service backend:
```bash
kubectl apply -f docs/example/manifests/split/traffic-split-v1.yaml
```

The counters should now be incrementing for the `Bookbuyer`, and `Bookstore-v1` applications:
- http://localhost:8080 - **bookbuyer**
- http://localhost:8081 - **bookstore-v1**

#### Allowing the Bookthief Application to access the Mesh
Currently the Bookthief application has not been authorized to participate in the service mesh communication. We will now uncomment out the lines in the [docs/example/manifests/access/traffic-access-v1.yaml](manifests/access/traffic-access-v1.yaml) to allow `bookthief` to communicate with `bookstore-v1`. Then, re-apply the manifest and watch the change in policy propagate.

Current TrafficTarget spec with commented `bookthief` kind:
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
kubectl apply -f docs/example/manifests/access/traffic-access-v1.yaml
```
The counter in the `Bookthief` window will start incrementing.
- http://localhost:8083 - **bookthief**

### Configure Traffic Split between Two Services

We will now demonstrate how to balance traffic between two Kubernetes services, commonly known as a traffic split. We will be splitting the traffic directed to the root `bookstore` service between the backends `bookstore-v1` service and `bookstore-v2` service. The bookstore-v1 service is currently deployed. We will deploy the bookstore-v2 service of the `Bookstore` app. Once the bookstore-v2 service is deployed we will then apply the updated `TrafficSplit` configuration between the two services.

#### Split Traffic to v2 of Bookstore

Deploy an SMI TrafficTarget policy to allow `bookbuyer` and `bookthief` to access the `bookstore-v2` service:
```bash
kubectl apply -f docs/example/manifests/access/traffic-access-v2.yaml
```

Browse to http://localhost:8082. You should see the `bookstore-v2` heading in your browser window. **NOTE** Please exit and restart the `./scripts/port-forward-all.sh` script in order to access v2 of Bookstore.

After restarting the port forwarding script, you should now be able to access the `bookstore-v2` application at http://localhost:8082. The count for the books sold should remain at 0, this is because the current traffic split policy is currently weighted 100% for `bookstore-v1`. You can verify the traffic split policy by running the following and viewing the **Backends** properties:
```
kubectl describe trafficsplit bookstore-split -n bookstore
```

#### Update Traffic Split

Update the SMI TrafficSplit policy for `bookstore` Service configuring all traffic to go to `bookstore-v2`:
```bash
kubectl apply -f docs/example/manifests/split/traffic-split-v2.yaml
```

Wait for the changes to propagate and observe the counters increment for bookstore-v2 in your browser windows. Modify the `weight` fields in [manifests/split-v2/traffic-split-v2.yaml](manifests/split-v2/traffic-split-v2.yaml) and re-apply changes to experiment.

- http://localhost:8082 - **bookstore-v2**

## Permissive Traffic Policy Mode

In permissive traffic policy mode, application connectivity within the mesh is automatically configured by `osm-controller`. It can be enabled in the following ways.

1. During install using `osm` cli:
```bash
osm install --enable-permissive-traffic-policy
```

2. Post install by updating the `osm-config` ConfigMap in the control plane's namespace (`osm-system` by default)
```bash
# Replace osm-system with osm-controller's namespace if using a non default namespace
kubectl patch ConfigMap osm-config -n osm-system -p '{"data":{"permissive_traffic_policy_mode":"true"}}' --type=merge
```

### Deploy a service with permissive traffic policy mode enabled

Before proceeding, [verify the traffic policy mode](#verify-the-traffic-policy-mode) and ensure the `permissive_traffic_policy_mode` key is set to `true` in the `osm-config` ConfigMap. Refer to the section above to enable permissive traffic policy mode.

For this demo, we will use the root `bookstore` service to demonstrate permissive traffic policy mode.
Deploy a new `bookstore` application backing the root `bookstore` service. The `bookstore` deployment has been configured with an identity of `bookstore-v1` and labels to match the root `bookstore` service for the purpose of this demo:
```bash
kubectl apply -f docs/example/manifests/apps/bookstore-root-service.yaml
kubectl apply -f docs/example/manifests/apps/bookstore-root-deployment.yaml
```

*Note: Unlike the SMI traffic policy mode demo where the root `bookstore` service does not need to be backed by a pod, permissive traffic policy mode requires every service that needs to be reachable to be backed by a pod.*

The counter in the `bookbuyer` and `bookthief` window for the books bought and stolen respectively from `bookstore v1` should now be incrementing:
- http://localhost:8080 - **bookbuyer**
- http://localhost:8083 - **bookthief**

The `bookbuyer` and `bookthief` applicatios are able to purchase books from the newly deployed `bookstore` service because permissive traffic policy mode is enabled, thereby allowing connectivity between applications without the need for SMI traffic policies.

This can be demonstrated further by disabling permissive traffic policy mode and verifying that the counter for books bought from `bookstore-v1` is not incrementing anymore:
```bash
# Replace osm-system with osm-controller's namespace if using a non default namespace
kubectl patch ConfigMap osm-config -n osm-system -p '{"data":{"permissive_traffic_policy_mode":"false"}}' --type=merge
```

*Note:
1. When you disable permissive traffic policy mode, SMI traffic policy mode is implicitly enabled, so if counters for the books are incrementing then it is because some SMI policies have been applied previously to allow such traffic.
1. In permissive traffic policy mode, a Kubernetes service must be created even for client pods that do not expose a service.

## Traffic Encryption

All traffic is encrypted via mTLS regardless of whether you're using access control policies or have enabled permissive traffic policy mode.

## Inspect Dashboards
OSM can be configured to deploy Grafana dashboards using the `--deploy-grafana` flag in `osm install`. **NOTE** If you still have the additional terminal still running the `./scripts/port-forward-all.sh` script, go ahead and `CTRL+C` to terminate the port forwarding. The `osm dashboard` port redirection will not work simultaneously with the port forwarding script still running. The `osm dashboard` can be viewed with the following command:
```bash
$ osm dashboard
```
Simply navigate to http://localhost:3000 to access the Grafana dashboards. The default user name is `admin` and the default password is `admin`. On the Grafana homepage click on the **Home** icon, you will see a folders containing dashboards for both OSM Control Plan and OSM Data Plane.

[1]: https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-access/v1alpha2/traffic-access.md
[2]: https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-specs/v1alpha4/traffic-specs.md
[3]: https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-split/v1alpha2/traffic-split.md
