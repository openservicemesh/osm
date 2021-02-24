---
title: "OSM Manual Demo"
description: "The manual demo is a step-by-step walkthrough set of instruction of the automated demo."
type: docs
aliases: ["OSM Manaual Demo"]
weight: 2
---

# How to Run the OSM Manual Demo

The OSM Manual Install Demo Guide is designed to quickly allow you to demo and experience the OSM mesh.

## Configure Prerequisites

- Kubernetes cluster running Kubernetes v1.15.0 or greater
- Have `kubectl` CLI installed - [Install and Set Up Kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- kubectl current context is configured for the target cluster install
  - `kubectl config current-context`
- Have a local clone of the OSM GitHub Repo
  - `git clone https://github.com/openservicemesh/osm.git`
  - `cd osm`

## Install OSM CLI

Use the [installation guide](/docs/content/docs/installation_guide.md) to install the `osm` cli.

## Install OSM Control Plane

For the purpose of this demo, it is recommended to install OSM with [permissive traffic policy mode](#permissive-traffic-policy-mode) enabled. By default, OSM will install with permissive traffic policy mode disabled and [SMI Traffic Policy Mode](#smi-traffic-policy-mode) enabled.

<<<<<<< HEAD:docs/example/README.md
*Note: By default, `osm` CLI does not enable Prometheus, Grafana, and Jaeger as a part of control plane installation.*
=======
_Note: By default, `osm` CLI does not enable Prometheus, Grafana, and Jaegar as a part of control plane installation._
>>>>>>> Began structure of site navigation:docs/content/docs/install/manual_demo.md

1.  Install OSM in permissive traffic policy mode:

    ```bash
    osm install --enable-permissive-traffic-policy
    ```

1.  Install OSM in SMI traffic policy mode:

    ```bash
    osm install
    ```

<<<<<<< HEAD:docs/example/README.md
1. To enable Prometheus, Grafana and Jaeger, use their respective flags
=======
1.  To enable Prometheus and Grafana, use their respective flags
>>>>>>> Began structure of site navigation:docs/content/docs/install/manual_demo.md
    ```bash
    osm install --deploy-prometheus --deploy-grafana --deploy-jaeger
    ```
    See the [observability documentation](/docs/content/docs/patterns/observability/_index.md) for more details.

## Deploying the Bookstore Demo Applications

The demo consists of the following resources:

- `bookbuyer` application that makes requests to the `bookstore` service to buy books
- `bookthief` application that makes requests to the `bookstore` service to steal books
- `bookstore` service that allows clients to purchase books
- `bookwarehouse` service that the `bookstore` service reaches out to restock books

When we demonstrate traffic splitting using SMI Traffic Split, we will deploy two additional services:

- `bookstore-v1` service representing version v1 of the `bookstore` service
- `bookstore-v2` service representing version v2 of the `bookstore` service

The `bookbuyer`, `bookthief`, `bookstore`, and `bookwarehouse` demo applications will be installed in their respective Kubernetes Namespaces. In order for these applications to be injected with a Envoy sidecar automatically, we must add the Namespaces to be monitored by the mesh.

### Create the Bookstore Application Namespaces

```bash
for i in bookstore bookbuyer bookthief bookwarehouse; do kubectl create ns $i; done
```

### Onboard the Namespaces to the OSM Mesh and enable sidecar injection on the namespaces

```bash
osm namespace add bookstore bookbuyer bookthief bookwarehouse
```

### Deploy the Bookstore Application

Deploy `bookbuyer`, `bookthief`, `bookstore`, `bookwarehouse` applications:

```bash
kubectl apply -f docs/example/manifests/apps/bookbuyer.yaml
kubectl apply -f docs/example/manifests/apps/bookthief.yaml
kubectl apply -f docs/example/manifests/apps/bookstore.yaml
kubectl apply -f docs/example/manifests/apps/bookwarehouse.yaml
```

### Checkpoint: What Got Installed?

A Kubernetes Service, Deployment, and ServiceAccount for applications `bookbuyer`, `bookthief`, `bookstore` and `bookwarehouse`.

To view these resources on your cluster, run the following commands:

```
kubectl get svc --all-namespaces
kubectl get deployment --all-namespaces
```

### View the Application UIs

We will now setup client port forwarding, so we can access the services in the Kubernetes cluster. It is best to start a new terminal session for running the port forwarding script to maintain the port forwarding session, while using the original terminal to continue to issue commands. The port-forward-all.sh script will look for a `".env"` file for variables. The `".env"` creates the necessary variables that target the previously created namespaces. We will use the reference .env.examples file and then run the port forwarding script.

In a new terminal session, run the following commands to enable port forwarding into the Kubernetes cluster.

```bash
cp .env.example .env
./scripts/port-forward-all.sh
```

_Note: To override the default ports, prefix the `BOOKBUYER_LOCAL_PORT`, `BOOKSTORE_LOCAL_PORT`, `BOOKSTOREv1_LOCAL_PORT`, `BOOKSTOREv2_LOCAL_PORT`, and/or `BOOKTHIEF_LOCAL_PORT` variable assignments to the `port-forward` scripts. For example:_

```bash
BOOKBUYER_LOCAL_PORT=7070 BOOKSTOREv1_LOCAL_PORT=7071 BOOKSTOREv2_LOCAL_PORT=7072 BOOKTHIEF_LOCAL_PORT=7073 BOOKSTORE_LOCAL_PORT=7074 ./scripts/port-forward-all.sh
```

In a browser, open up the following urls:

- http://localhost:8080 - **bookbuyer**
- http://localhost:8083 - **bookthief**
- http://localhost:8084 - **bookstore**
- http://localhost:8081 - **bookstore-v1**
  - _Note: This page will not be available at this time in the demo. This will become available during the SMI Traffic Split configuration set up_
- http://localhost:8082 - **bookstore-v2**
  - _Note: This page will not be available at this time in the demo. This will become available during the SMI Traffic Split configuration set up_

Position the windows so that you can see all four at the same time. The header at the top of the webpage indicates the application and version.

## Traffic Encryption

All traffic is encrypted via mTLS regardless of whether you're using access control policies or have enabled permissive traffic policy mode.

## Traffic Policy Modes

Once the applications are up and running, they can interact with each other using [permissive traffic policy mode](#permissive-traffic-policy-mode) or [SMI traffic policy mode](#smi-traffic-policy-mode). In permissive traffic policy mode, traffic between application services is automatically configured by `osm-controller`, and SMI policies are not enforced. In the SMI policy mode, all traffic is denied by default unless explicitly allowed using a combination of SMI access and routing policies.

### Verify the Traffic Policy Mode

Check whether permissive traffic policy mode is enabled or not by retrieving the value for the `permissive_traffic_policy_mode` key in the `osm-config` ConfigMap.

```bash
# Replace osm-system with osm-controller's namespace if using a non default namespace
kubectl get configmap -n osm-system osm-config -o json | jq -r '.data["permissive_traffic_policy_mode"]'
# Output:
# false: permissive traffic policy mode is disabled, SMI policy mode is enabled
# true: permissive traffic policy mode is enabled, SMI policy mode is disabled
```

The following sections demonstrate using OSM with [permissive traffic policy mode](#permissive-traffic-policy-mode) and [SMI Traffic Policy Mode](#smi-traffic-policy-mode).

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

### Verify traffic in permissive traffic policy mode

Before proceeding, [verify the traffic policy mode](#verify-the-traffic-policy-mode) and ensure the `permissive_traffic_policy_mode` key is set to `true` in the `osm-config` ConfigMap. Refer to the section above to enable permissive traffic policy mode.

In step [Deploy the Bookstore Application](#deploy-the-bookstore-application), we have already deployed the applications needed to verify traffic flow in permissive traffic policy mode. The `bookstore` service we previously deployed is encoded with an identity of `bookstore-v1` for demo purpose, as can be seen in the Deployment spec [docs/example/manifests/apps/bookstore.yaml](manifests/apps/bookstore.yaml). The identity reflects which counter increments in the `bookbuyer` and `bookthief` UI, and the identity displayed in the `bookstore` UI.

The counter in the `bookbuyer`, `bookthief` UI for the books bought and stolen respectively from `bookstore v1` should now be incrementing:

- http://localhost:8080 - **bookbuyer**
- http://localhost:8083 - **bookthief**

The counter in the `bookstore` UI for the books sold should also be incrementing:

- http://localhost:8084 - **bookstore**

The `bookbuyer` and `bookthief` applications are able to buy and steal books respectively from the newly deployed `bookstore` service with identity `bookstore-v1` because permissive traffic policy mode is enabled, thereby allowing connectivity between applications without the need for SMI traffic policies.

This can be demonstrated further by disabling permissive traffic policy mode and verifying that the counter for books bought from `bookstore-v1` is not incrementing anymore:

```bash
# Replace osm-system with osm-controller's namespace if using a non default namespace
kubectl patch ConfigMap osm-config -n osm-system -p '{"data":{"permissive_traffic_policy_mode":"false"}}' --type=merge
```

\*Note:

1. When you disable permissive traffic policy mode, SMI traffic policy mode is implicitly enabled, so if counters for the books are incrementing then it could be because some SMI policies have been applied previously to allow such traffic.
1. In permissive traffic policy mode, a Kubernetes service must be created even for client pods that do not expose a service.

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

### Deploy bookstore v1 and v2 services

To demonstrate usage of SMI traffic access and split policies, we will deploy now deploy two versions of the bookstore application - `bookstore-v1` and `bookstore-v2`.

```bash
kubectl apply -f docs/example/manifests/apps/bookstore-v1.yaml
kubectl apply -f docs/example/manifests/apps/bookstore-v2.yaml
```

Wait for the `bookstore-v1` and `bookstore-v2` pods to be running in the `bookstore` namespace. Next, exit and restart the `./scripts/port-forward-all.sh` script in order to access v1 and v2 of bookstore.

- http://localhost:8081 - **bookstore-v1**
- http://localhost:8082 - **bookstore-v2**

A simple topology view of the Bookstore application now looks like the following:
![Bookstore Application Topology](/img/book-thief-app-topology.jpg "Bookstore Application Topology")

### Deploy SMI Access Control Policies

At this point, applications do not have access to each other because no access control policies have been applied. Confirm this by verifying that none of the counters in the `bookbuyer`, `bookthief`, `bookstore-v1`, and `bookstore-v2` UI are incrementing.

Apply the [SMI Traffic Target][1], [SMI Traffic Specs][2], and [SMI TrafficSplit][3] resources to define access control and routing policies for the applications to communicate:

```bash
# Deploy SMI TrafficTarget and HTTPRouteGroup policy
kubectl apply -f docs/example/manifests/access/traffic-access-v1.yaml
```

_Note: At the moment, you must configure a TrafficSplit resource to get your applications set up correctly for inbound traffic because it helps us properly configure the dataplane. We are working on removing the need for this entirely._ [#1370](https://github.com/openservicemesh/osm/issues/1370)

Deploy the SMI traffic split policy to direct 100 percent of the traffic sent to the root `bookstore` service to the `bookstore-v1` service backend:

```bash
kubectl apply -f docs/example/manifests/split/traffic-split-v1.yaml
```

The counters should now be incrementing for the `bookbuyer`, and `bookstore-v1` applications:

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

Updated TrafficTarget spec with uncommented `bookthief` kind:

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

The counter in the `bookthief` window will start incrementing.

- http://localhost:8083 - **bookthief**

### Configure Traffic Split between Two Services

We will now demonstrate how to balance traffic between two Kubernetes services, commonly known as a traffic split. We will be splitting the traffic directed to the root `bookstore` service between the backends `bookstore-v1` service and `bookstore-v2` service.

#### Split Traffic to v2 of Bookstore

Deploy an SMI TrafficTarget policy to allow `bookbuyer` and `bookthief` to access the `bookstore-v2` service:

```bash
kubectl apply -f docs/example/manifests/access/traffic-access-v2.yaml
```

Browse to http://localhost:8082. You should see the `bookstore-v2` heading in your browser window.

The count for the books sold should remain at 0, this is because the current traffic split policy is currently weighted 100 for `bookstore-v1`. You can verify the traffic split policy by running the following and viewing the **Backends** properties:

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

## Inspect Dashboards

OSM can be configured to deploy Grafana dashboards using the `--deploy-grafana` flag in `osm install`. **NOTE** If you still have the additional terminal still running the `./scripts/port-forward-all.sh` script, go ahead and `CTRL+C` to terminate the port forwarding. The `osm dashboard` port redirection will not work simultaneously with the port forwarding script still running. The `osm dashboard` can be viewed with the following command:

```bash
$ osm dashboard
```

Simply navigate to http://localhost:3000 to access the Grafana dashboards. The default user name is `admin` and the default password is `admin`. On the Grafana homepage click on the **Home** icon, you will see a folders containing dashboards for both OSM Control Plan and OSM Data Plane.

[1]: https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-access/v1alpha2/traffic-access.md
[2]: https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-specs/v1alpha4/traffic-specs.md
[3]: https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-split/v1alpha2/traffic-split.md
