# Upgrade Guide

This guide describes how to upgrade Open Service Mesh (OSM).

## Prerequisites
- Kubernetes cluster with OSM installed
- The [helm 3 CLI](https://helm.sh/docs/intro/install/)

## Upgrade OSM using Helm
Use the `helm` CLI to upgrade the OSM control plane from a Kubernetes cluster.

Run `helm upgrade`. 
```console
# Add steps for helm upgrade
```

Run `helm upgrade --help` for more options.

## CRD Upgrades
Please check the `CRD Updates` section of the [release notes](https://github.com/openservicemesh/osm/releases) to see if additional steps are required to update the CRDs used by OSM. If the new release does contain updates to the CRDs, it is required to first delete existing CRDs and the associated Custom Resources prior to upgrading.

In the `./scripts/cleanup` directory we have included a helper script to delete those CRDs and Custom Resources: `./scripts/osm-cleanup.sh`

## Upgrade support
Upgrades are supported between consecutive minor versions. Skipping minor versions is not supported.

## Resource availability during upgrade
Since upgrades include redeploying the osm-controller with the new version, there may be some downtime of the controller. This means that there will be a delay in processing new SMI resources. 

However, already existing SMI resources will be unaffected (assuming [CRD Upgrades](#CRD-Upgrades) are not needed). This means that the data plane (which includes the Envoy sidecar configs) will also be unaffected by upgrading. 