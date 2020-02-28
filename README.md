# SMC

[![build](https://github.com/deislabs/smc/workflows/Go/badge.svg)](https://github.com/deislabs/smc/actions?query=workflow%3AGo)
[![report](https://goreportcard.com/badge/github.com/deislabs/smc)](https://goreportcard.com/report/github.com/deislabs/smc)
[![codecov](https://codecov.io/gh/deislabs/smc/branch/master/graph/badge.svg)](https://codecov.io/gh/deislabs/smc)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/deislabs/smc/blob/master/LICENSE)
[![release](https://img.shields.io/github/release/deislabs/smc/all.svg)](https://github.com/deislabs/smc/releases)

The Service Mesh Controller (SMC) project is a light weight, envoy based service mesh for applications running in Kubernetes and on VMs. It works with Envoy proxies configured as side-car containers and continuously programs them to implement Service Mesh Interface(SMI) policies. It provides the following key benefits
1. Native support for Virtual Machines. Can be easily extended to support Serverless workloads also. 
2. Compatible with Service Mesh Interface specification. Users can express Service Mesh policies through SMI
3. Provides declarative APIs to add and remove Kubernetes Services and VMs in a mesh. Supports Hybrid Meshes comprising of K8S services, VMs and other types of compute instances. 
4. Provides auto-injection of Envoy proxy in Kubernetes services and Virtual Machines when added to the mesh
5. Provides a pluggable interface to integrate with external certificate management services/solutions 

Note: This project is a work in progress. See the [demo instructions](demo/README.md) to get a sense of what we've accomplished and are working on.


## SMC Design
Read more about the high level goals, design and architecture [here](DESIGN.md).
