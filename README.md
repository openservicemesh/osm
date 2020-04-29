# Open Service Mesh (OSM)

[![build](https://github.com/open-service-mesh/osm/workflows/Go/badge.svg)](https://github.com/open-service-mesh/osm/actions?query=workflow%3AGo)
[![report](https://goreportcard.com/badge/github.com/open-service-mesh/osm)](https://goreportcard.com/report/github.com/open-service-mesh/osm)
[![codecov](https://codecov.io/gh/open-service-mesh/osm/branch/master/graph/badge.svg)](https://codecov.io/gh/open-service-mesh/osm)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/open-service-mesh/osm/blob/master/LICENSE)
[![release](https://img.shields.io/github/release/open-service-mesh/osm/all.svg)](https://github.com/open-service-mesh/osm/releases)

The Open Service Mesh (OSM) project is a light weight, envoy based service mesh for applications running in Kubernetes and on VMs. It works with Envoy proxies configured as side-car containers and continuously programs them to implement Service Mesh Interface(SMI) policies. It provides the following key benefits
1. Native support for Virtual Machines. Can be easily extended to support Serverless workloads also.
2. Compatible with Service Mesh Interface specification. Users can express Service Mesh policies through SMI
3. Provides declarative APIs to add and remove Kubernetes Services and VMs in a mesh. Supports Hybrid Meshes comprising of K8S services, VMs and other types of compute instances.
4. Provides auto-injection of Envoy proxy in Kubernetes services and Virtual Machines when added to the mesh
5. Provides a pluggable interface to integrate with external certificate management services/solutions

Note: This project is a work in progress. See the [demo instructions](demo/README.md) to get a sense of what we've accomplished and are working on.

## OSM Design

Read more about the high level goals, design and architecture [here](DESIGN.md).

## Managing Services Using OSM

#### On-boarding services to the OSM managed service mesh

To on-board a service to the OSM managed service mesh, OSM first needs to be configured to monitor the namespace the service belongs to. This can be done by labeling the namespace with the OSM ID as follows.
```
$ kubectl label namespace <namespace> openservicemesh.io/monitor=<osm-id>
```
The `osm-id` is a unique ID for an OSM instance generated during OSM install.

After a namespace is labeled for monitoring, new services deployed in a monitored namespace will be a part of the service mesh and OSM will perform automatic sidecar injection for newly created PODs in that namespace.

For a service to be a part of the service mesh, OSM requires a 1-to-1 mapping between a pod, a service and a service account. This means a pod in a service mesh must run a single service, and the service must be associated with a single service account.
This can be achieved using either of the following ways:
- A service running on a pod must have the same name as the service account of the pod, or
- A pod running a service whose name is not the same as the pod's service account name should be annoted with the name of the service

For example, if a service `bookstore` is running on a pod with service account `bookstore-svc-acc`, the pod must be annotated with the service name as `"openservicemesh.io/osm-service": "bookstore"`. If the pod's service account has the same name as the service running on it, `bookstore` in this example, then such an annotation is not required.

#### Disabling automatic sidecar injection
Since the sidecar is automatically injected for PODs belonging to a monitored namespace, PODs that are not a part of the service mesh but belong to a monitored namespace should be configured to not have the sidecar injected. This can be achieved by any of the following ways.

- Deploying PODs that are not a part of the service mesh in namespaces that are not monitored by OSM
- Explicitly annotating PODs with sidecar injection as disabled: `"openservicemesh.io/sidecar-injection": "disabled"`

#### Adding existing services to be managed by a new OSM instance
Currently OSM only supports automatic sidecar injection for newly created PODs. Thus, existing services will need to be enabled for monitoring as described above, and then the PODs will need to be redeployed. This workflow will be simplified once OSM supports manual sidecar injection.

#### Un-managing namespaces
To stop OSM from monitoring a namespace, remove the monitoring label from the namespace.
```
$ kubectl label namespace <namespace> openservicemesh.io/monitor-
```