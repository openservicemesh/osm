---
title: "OSM Demo: Split Traffic"
description: "This demo of OSM is a walkthrough of splitting traffic to two different versions of a service."
type: docs
aliases: ["OSM Split Traffic"]
weight: 2
release: 0.8.0
---


# Open Service Mesh Demo: Splitting Traffic


> Note: This demo is specific to OSM v0.8.0


This document builds on the [First Steps](first_steps.md) demo and will walk you through the steps to:
  - create a `bookstore-v2` **service**
  - deploy a `bookstore-v2` **pod** for the `bookstore-v2` service
  - apply [TrafficSplit SMI](https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-split/v1alpha3/traffic-split.md#traffic-split) policy to route traffic from `bookbuyer` to `bookstore` and `bookstore-v2` services.


## Prerequisites
This demo of OSM v0.8.0 requires:
  - a cluster running Kubernetes v1.15.0 or greater
  - a workstation capable of executing [Bash](https://en.wikipedia.org/wiki/Bash_(Unix_shell)) scripts
  - [The Kubernetes command-line tool](https://kubernetes.io/docs/tasks/tools/#kubectl) - `kubectl`
  - all resources from the [First Steps demo](first_steps.md)
    - 2 namespaces joined to the service mesh: `bookbuyer` and `bookstore`
    - 2 service accounts
    - 2 pods, `bookbuyer` and `bookstore`, in their respective namespaces


> If you have not walked through the [First Steps demo](first_steps.md) you are missing Kubernetes resources required by this document. The expandable section below has all necessary commands to quickly copy/paste and create the required components:

<details>
  <summary>Click to expand: steps from First Steps demo</summary>

Download OSM:
```bash
system=$(uname -s | tr '[:upper:]' '[:lower:]')
release=v0.8.0
curl -L https://github.com/openservicemesh/osm/releases/download/${release}/osm-${release}-${system}-amd64.tar.gz | tar -vxzf -
./${system}-amd64/osm version
mv ./${system}-amd64/osm ./

./osm install
```

Create Kubernetes resources and SMI Policies:
```bash
kubectl create namespace bookbuyer
kubectl create namespace bookstore

osm namespace add bookbuyer
osm namespace add bookstore

kubectl apply -f - <<EOF
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: bookbuyer
  namespace: bookbuyer
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
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: bookstore
  namespace: bookstore
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

Check `bookbuyer` logs for successful HTTP GET requests to `bookstore`:
```bash
kubectl logs -n bookbuyer bookbuyer -c bookbuyer -f | grep 'Status:'
```
</details>


## Install `bookstore-v2`

Create the `bookstore-v2` service:
```bash
kubectl apply -f - <<EOF
---
apiVersion: v1
kind: Service
metadata:
  name: bookstore-v2
  namespace: bookstore
  labels:
    app: bookstore
spec:
  selector:
    app: bookstore-v2
  ports:
  - port: 14001
EOF
```


Create the `bookstore-v2` pod:
```bash
kubectl apply -f - <<EOF
---
apiVersion: v1
kind: Pod
metadata:
  namespace: bookstore
  name: bookstore-v2
  labels:
    app: bookstore-v2
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
      value: bookstore-v2
EOF
```

View the Kubernetes resources created so far:

```bash
kubectl get pods -n bookstore

kubectl get services -n bookstore

kubectl get endpoints -n bookstore
```

<details>
  <summary>Click to expand: sample kubectl output</summary>

Pods:
```
NAME           READY   STATUS    RESTARTS   AGE
bookstore      2/2     Running   0          12h
bookstore-v2   2/2     Running   0          18s
```

Services:
```
NAME           TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)     AGE
bookstore      ClusterIP   10.0.22.160   <none>        14001/TCP   12h
bookstore-v2   ClusterIP   10.0.89.109   <none>        14001/TCP   25s
```

Endpoints:
```
NAME           ENDPOINTS            AGE
bookstore      10.240.1.222:14001   12h
bookstore-v2   10.240.1.249:14001   25s
```

</details>

---

## Apply TrafficSplit SMI Policy

The following
[TrafficSplit SMI](https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-split/v1alpha3/traffic-split.md#traffic-split)
policy will configure the Envoy on the `bookbuyer` to use both bookstore versions.
The line `weight: 50` indicates that 50% of the HTTP GET requests will be routed to `bookstore` and the other half to `bookstore-v2` pods.

```bash
kubectl apply -f - <<EOF
---
apiVersion: split.smi-spec.io/v1alpha2
kind: TrafficSplit
metadata:
  name: bookstore-split
  namespace: bookstore
spec:
  service: bookstore.bookstore # <root-service>.<namespace>
  backends:
  - service: bookstore
    weight: 50
  - service: bookstore-v2
    weight: 50
EOF
```

Take a look at the `bookbuyer` logs:
```bash
kubectl logs -n bookbuyer bookbuyer -c bookbuyer --tail=90 -f | grep 'Identity:'
```

You'll notice that every other response is from `bookstore-v2`.


Read more about OSM's traffic split implementation here. `//TODO(draychev): Link to TrafficSplit_...md doc`


# Summary
1. We installed a second version of the bookstore service: `bookstore-v2`
2. We applied a [TrafficSplit SMI](https://github.com/servicemeshinterface/smi-spec/blob/v0.6.0/apis/traffic-split/v1alpha3/traffic-split.md#traffic-split) policy
3. We saw requests from the `bookbuyer` pod alternate between the 2 bookstore pods: `bookstore` and `bookstore-v2`


# Next Steps
`//TODO(draychev): add a link to the next demo`
