---
title: "OSM Manual Demo Guide"
description: "The manual demo is a step-by-step walkthrough set of instruction of the automated demo."
type: docs
aliases: ["OSM Manaual Demo"]
weight: 2
---

# OSM Manual Demo Guide

The OSM Manual Install Demo Guide is a step by step set of instructions to quickly demo OSM's key features.

## Configure Prerequisites

- Kubernetes cluster running Kubernetes v1.15.0 or greater
- Have `kubectl` CLI installed - [Install and Set Up Kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- kubectl current context is configured for the target cluster install
  - `kubectl config current-context`
- Have a local clone of the OSM GitHub Repo
  - `git clone https://github.com/openservicemesh/osm.git`
  - `cd osm`

## Build or Download the OSM CLI

Use the [installation guide](../../install) to install the `osm` cli.

## Install OSM Control Plane

For the purpose of this demo, install OSM with [permissive traffic policy mode](#permissive-traffic-policy-mode) enabled via the `--enable-permissive-traffic-policy` flag. By default, OSM will install with permissive traffic policy mode disabled and [SMI Traffic Policy Mode](#smi-traffic-policy-mode) enabled. Also by default, `osm` CLI does not enable Prometheus, Grafana, and Jaeger as a part of control plane installation.

Install OSM in permissive traffic policy mode with these features enabled:

```bash
osm install --enable-permissive-traffic-policy --deploy-prometheus --deploy-grafana --deploy-jaeger
```

See the [observability documentation](../../patterns/observability/_index.md) for more details about using Prometheus, Grafana, and Jaeger with OSM.

### OpenShift
For details on how to install OSM on OpenShift, refer to the [installation guide](../#openshift)

## Deploy the Bookstore Demo Applications

This demo consists of the following applications:

- `bookbuyer` application that makes requests to `bookstore` to buy books
- `bookthief` application that makes requests to `bookstore` to steal books
- `bookstore` application that receives requests from clients to purchase books and makes requests to `bookwarehouse` to restock books
- `bookwarehouse` application that receives requests from `bookstore` to restock books

When we demonstrate traffic splitting using SMI Traffic Split, we will deploy an additional application:

- `bookstore-v2` application representing version v2 of the `bookstore` application

The `bookbuyer`, `bookthief`, `bookstore`, and `bookwarehouse` demo applications will be installed in their respective Kubernetes Namespaces. For OSM to work, each application Pod must contain an Envoy proxy as a sidecar container. OSM can automatically inject an Envoy proxy sidecar into application Pods. Use the `osm` CLI to add Kubernetes Namespaces to monitor and inject proxies into using the `osm namespace add` command. Once the Namespace is added, OSM will monitor the Namespace for new Pods and automatically inject the Envoy proxy as a sidecar into each Pod created in the Namespace.

### Create the Bookstore Application Namespaces

```bash
for i in bookstore bookbuyer bookthief bookwarehouse; do kubectl create ns $i; done
```

### Onboard the Namespaces to the OSM Mesh and enable sidecar injection on the namespaces

```bash
osm namespace add bookstore bookbuyer bookthief bookwarehouse
```

### Create the Kubernetes resources for the bookstore demo applications

Deploy `bookbuyer`, `bookthief`, `bookstore`, `bookwarehouse` applications:

```bash
kubectl apply -f https://raw.githubusercontent.com/openservicemesh/osm/main/docs/example/manifests/apps/bookbuyer.yaml
kubectl apply -f https://raw.githubusercontent.com/openservicemesh/osm/main/docs/example/manifests/apps/bookthief.yaml
kubectl apply -f https://raw.githubusercontent.com/openservicemesh/osm/main/docs/example/manifests/apps/bookstore.yaml
kubectl apply -f https://raw.githubusercontent.com/openservicemesh/osm/main/docs/example/manifests/apps/bookwarehouse.yaml
```

### Checkpoint: What Got Installed?

A Kubernetes Service, Deployment, and Service Account for applications `bookbuyer`, `bookthief`, `bookstore` and `bookwarehouse`.

To view these resources on your cluster, run the following commands:

```bash
kubectl get svc --all-namespaces
kubectl get deployment --all-namespaces
```

In addition to Kubernetes Services and Deployments, a [Kubernetes Service Account](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/) was also created for each Deployment. The Service Account services as the application's identity which will be used later in the demo to create service to service access control policies.

### View the Application UIs

Set up client port forwarding with the following steps to access the applications in the Kubernetes cluster. It is best to start a new terminal session for running the port forwarding script to maintain the port forwarding session, while using the original terminal to continue to issue commands. The port-forward-all.sh script will look for a `.env` file for environment variables needed to run the script. The `.env` creates the necessary variables that target the previously created namespaces. We will use the reference `.env.example` file and then run the port forwarding script.

In a new terminal session, run the following commands to enable port forwarding into the Kubernetes cluster from the root of the project directory.

```bash
cp .env.example .env
./scripts/port-forward-all.sh
```

_Note: To override the default ports, prefix the `BOOKBUYER_LOCAL_PORT`, `BOOKSTORE_LOCAL_PORT`, `BOOKSTOREv1_LOCAL_PORT`, `BOOKSTOREv2_LOCAL_PORT`, and/or `BOOKTHIEF_LOCAL_PORT` variable assignments to the `port-forward` scripts. For example:_

```bash
BOOKBUYER_LOCAL_PORT=7070 BOOKSTOREv1_LOCAL_PORT=7071 BOOKSTOREv2_LOCAL_PORT=7072 BOOKTHIEF_LOCAL_PORT=7073 BOOKSTORE_LOCAL_PORT=7074 ./scripts/port-forward-all.sh
```

In a browser, open up the following urls:

- [http://localhost:8080](http://localhost:8080) - **bookbuyer**
- [http://localhost:8083](http://localhost:8083) - **bookthief**
- [http://localhost:8084](http://localhost:8084) - **bookstore**
- [http://localhost:8082](http://localhost:8082) - **bookstore-v2**
  - _Note: This page will not be available at this time in the demo. This will become available during the SMI Traffic Split configuration set up_

Position the windows so that you can see all four at the same time. The header at the top of the webpage indicates the application and version.

## Traffic Encryption

All traffic is encrypted via mTLS regardless of whether you're using access control policies or have enabled permissive traffic policy mode.

## Traffic Policy Modes

Once the applications are up and running, they can interact with each other using [permissive traffic policy mode](#permissive-traffic-policy-mode) or [SMI traffic policy mode](#smi-traffic-policy-mode). In permissive traffic policy mode, traffic between application services is automatically configured by `osm-controller`, and SMI policies are not enforced. In the SMI policy mode, all traffic is denied by default unless explicitly allowed using a combination of SMI access and routing policies.

### How to Check Traffic Policy Mode

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

1. During install using `osm` CLI:

```bash
osm install --enable-permissive-traffic-policy
```

1. Post install by updating the `osm-config` ConfigMap in the control plane's namespace (`osm-system` by default)

```bash
osm mesh upgrade --enable-permissive-traffic-policy=true
```

### Verify OSM is in permissive traffic policy mode

Before proceeding, [verify the traffic policy mode](#verify-the-traffic-policy-mode) and ensure the `permissive_traffic_policy_mode` key is set to `true` in the `osm-config` ConfigMap. Refer to the section above to enable permissive traffic policy mode.

In step [Deploy the Bookstore Application](#deploy-the-bookstore-application), we have already deployed the applications needed to verify traffic flow in permissive traffic policy mode. The `bookstore` service we previously deployed is encoded with an identity of `bookstore-v1` for demo purpose, as can be seen in the [Deployment's manifest](https://raw.githubusercontent.com/openservicemesh/osm/main/docs/example/manifests/apps/bookstore.yaml). The identity reflects which counter increments in the `bookbuyer` and `bookthief` UI, and the identity displayed in the `bookstore` UI.

The counter in the `bookbuyer`, `bookthief` UI for the books bought and stolen respectively from `bookstore v1` should now be incrementing:

- [http://localhost:8080](http://localhost:8080) - **bookbuyer**
- [http://localhost:8083](http://localhost:8083) - **bookthief**

The counter in the `bookstore` UI for the books sold should also be incrementing:

- [http://localhost:8084](http://localhost:8084) - **bookstore**

The `bookbuyer` and `bookthief` applications are able to buy and steal books respectively from the newly deployed `bookstore` application because permissive traffic policy mode is enabled, thereby allowing connectivity between applications without the need for SMI traffic access policies.

This can be demonstrated further by disabling permissive traffic policy mode and verifying that the counter for books bought from `bookstore` is not incrementing anymore:

```bash
osm mesh upgrade --enable-permissive-traffic-policy=false
```

_Note: When you disable permissive traffic policy mode, SMI traffic access mode is implicitly enabled. If counters for the books are incrementing then it could be because some SMI Traffic Access policies have been applied previously to allow such traffic._

## SMI Traffic Policy Mode

SMI traffic policies can be used for the following:

1. SMI access control policies to authorize traffic access between service identities
1. SMI traffic specs policies to define routing rules to associate with access control policies
1. SMI traffic split policies to direct client traffic to multiple backends based on weights

The following sections describe how to leverage each of these policies to enforce fine grained control over traffic flowing within the service mesh. Before proceeding, [verify the traffic policy mode](#verify-the-traffic-policy-mode) and ensure the `permissive_traffic_policy_mode` key is set to `false` in the `osm-config` ConfigMap.

SMI traffic policy mode can be enabled by disabling permissive traffic policy mode:

```bash
osm mesh upgrade --enable-permissive-traffic-policy=false
```

### Deploy SMI Access Control Policies

At this point, applications do not have access to each other because no access control policies have been applied. Confirm this by verifying that none of the counters in the `bookbuyer`, `bookthief`, `bookstore`, and `bookstore-v2` UI are incrementing.

Apply the [SMI Traffic Target][1] and [SMI Traffic Specs][2] resources to define access control and routing policies for the applications to communicate:

```bash
# Deploy SMI TrafficTarget and HTTPRouteGroup policy
kubectl apply -f https://raw.githubusercontent.com/openservicemesh/osm/main/docs/example/manifests/access/traffic-access-v1.yaml
```

The counters should now be incrementing for the `bookbuyer`, and `bookstore` applications:

- [http://localhost:8080](http://localhost:8080) - **bookbuyer**
- [http://localhost:8084](http://localhost:8084) - **bookstore**

Note that the counter is _not_ incrementing for the `bookthief` application:

- [http://localhost:8083](http://localhost:8083) - **bookthief**

That is because the SMI Traffic Target SMI HTTPRouteGroup resources deployed only allow `bookbuyer` to communicate with the `bookstore`.

#### Allowing the Bookthief Application to access the Mesh

Currently the Bookthief application has not been authorized to participate in the service mesh communication. We will now uncomment out the lines in the [docs/example/manifests/access/traffic-access-v1.yaml](https://raw.githubusercontent.com/openservicemesh/osm/main/docs/example/manifests/access/traffic-access-v1.yaml) to allow `bookthief` to communicate with `bookstore`. Then, re-apply the manifest and watch the change in policy propagate.

Current TrafficTarget spec with commented `bookthief` kind:

```yaml
kind: TrafficTarget
apiVersion: access.smi-spec.io/v1alpha3
metadata:
  name: bookstore-v1
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

```yaml
kind: TrafficTarget
apiVersion: access.smi-spec.io/v1alpha3
metadata:
 name: bookstore-v1
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
kubectl apply -f https://raw.githubusercontent.com/openservicemesh/osm/main/docs/example/manifests/access/traffic-access-v1-allow-bookthief.yaml
```

The counter in the `bookthief` window will start incrementing.

- [http://localhost:8083](http://localhost:8083) - **bookthief**

Comment out the bookthief source lines in the Traffic Target object and re-apply the Traffic Access policies:

```bash
# Re-apply original SMI TrafficTarget and HTTPRouteGroup resources
kubectl apply -f https://raw.githubusercontent.com/openservicemesh/osm/main/docs/example/manifests/access/traffic-access-v1.yaml
```

The counter in the `bookthief` window will start incrementing.

- [http://localhost:8083](http://localhost:8083) - **bookthief**

### Configure Traffic Split between Two Services

We will now demonstrate how to balance traffic between two Kubernetes services, commonly known as a traffic split. We will be splitting the traffic directed to the root `bookstore` service between the backends `bookstore` service and `bookstore-v2` service.

### Deploy bookstore v2 application

To demonstrate usage of SMI traffic access and split policies, we will deploy now deploy version v2 of the bookstore application (`bookstore-v2`):

```bash
# Contains the bookstore-v2 Kubernetes Service, Service Account, Deployment and SMI Traffic Target resource to allow
# bookbuyer to communicate with `bookstore-v2` pods
kubectl apply -f https://raw.githubusercontent.com/openservicemesh/osm/main/docs/example/manifests/apps/bookstore-v2.yaml
```

Wait for the `bookstore-v2` pod to be running in the `bookstore` namespace. Next, exit and restart the `./scripts/port-forward-all.sh` script in order to access v2 of bookstore.

- [http://localhost:8082](http://localhost:8082) - **bookstore-v2**

The counter should _not_ be incrementing because no traffic is flowing yet to the `bookstore-v2` service.

#### Create SMI Traffic Split

Deploy the SMI traffic split policy to direct 100 percent of the traffic sent to the root `bookstore` service to the `bookstore` service backend:

```bash
kubectl apply -f https://raw.githubusercontent.com/openservicemesh/osm/main/docs/example/manifests/split/traffic-split-v1.yaml
```

_Note: The root service can be any Kubernetes service. It does not have any label selectors. It also doesn't need to overlap with any of the Backend services specified in the Traffic Split resource. The root service can be referred to in the SMI Traffic Split resource as the name of the service with or without the `.<namespace>` suffix._

The count for the books sold from the `bookstore-v2` browser window should remain at 0. This is because the current traffic split policy is currently weighted 100 for `bookstore` in addition to the fact that `bookbuyer` is sending traffic to the `bookstore` service and no application is sending requests to the `bookstore-v2` service. You can verify the traffic split policy by running the following and viewing the **Backends** properties:

```bash
kubectl describe trafficsplit bookstore-split -n bookstore
```

#### Split Traffic to Bookstore v2

Update the SMI Traffic Split policy to direct 50 percent of the traffic sent to the root `bookstore` service to the `bookstore` service and 50 perfect to `bookstore-v2` service by adding the `bookstore-v2` backend to the spec and modifying the weight fields.

```bash
kubectl apply -f https://raw.githubusercontent.com/openservicemesh/osm/main/docs/example/manifests/split/traffic-split-50-50.yaml
```

Wait for the changes to propagate and observe the counters increment for `bookstore` and `bookstore-v2` in your browser windows. Both
counters should be incrementing:

- [http://localhost:8084](http://localhost:8084) - **bookstore**
- [http://localhost:8082](http://localhost:8082) - **bookstore-v2**

#### Split All Traffic to Bookstore v2

Update the SMI TrafficSplit policy for `bookstore` Service configuring all traffic to go to `bookstore-v2`:

```bash
kubectl apply -f https://raw.githubusercontent.com/openservicemesh/osm/main/docs/example/manifests/split/traffic-split-v2.yaml
```

Wait for the changes to propagate and observe the counters increment for `bookstore-v2` and freeze for `bookstore` in your
browser windows:

- [http://localhost:8082](http://localhost:8082) - **bookstore-v2**
- [http://localhost:8083](http://localhost:8084) - **bookstore**

Now, all traffic directed to the `bookstore` service is flowing to `bookstore-v2`.

## Inspect Dashboards

OSM can be configured to deploy Grafana dashboards using the `--deploy-grafana` flag in `osm install`. **NOTE** If you still have the additional terminal still running the `./scripts/port-forward-all.sh` script, go ahead and `CTRL+C` to terminate the port forwarding. The `osm dashboard` port redirection will not work simultaneously with the port forwarding script still running. The `osm dashboard` can be viewed with the following command:

```bash
osm dashboard
```

Simply navigate to http://localhost:3000 to access the Grafana dashboards. The default user name is `admin` and the default password is `admin`. On the Grafana homepage click on the **Home** icon, you will see a folders containing dashboards for both OSM Control Plan and OSM Data Plane.

[1]: https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-access/v1alpha2/traffic-access.md
[2]: https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-specs/v1alpha4/traffic-specs.md
[3]: https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-split/v1alpha2/traffic-split.md
