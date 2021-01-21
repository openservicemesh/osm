# Upgrade Guide

This guide describes how to upgrade the Open Service Mesh (OSM) control plane.

## Prerequisites
- Kubernetes cluster with the OSM control plane installed
- The [helm 3 CLI](https://helm.sh/docs/intro/install/) 
  - The helm CLI must be used until osm CLI upgrade support is implemented.

## Resource availability during upgrade
Since upgrades include redeploying the osm-controller with the new version, there may be some downtime of the controller. This means that there will be a delay in processing new SMI resources. 

However, already existing SMI resources will be unaffected (assuming [CRD Upgrades](#CRD-Upgrades) are not needed). This means that the data plane (which includes the Envoy sidecar configs) will also be unaffected by upgrading. 

Data plane interruptions are expected if the upgrade includes CRD changes.

## CRD Upgrades
Please check the `CRD Updates` section of the [release notes](https://github.com/openservicemesh/osm/releases) to see if additional steps are required to update the CRDs used by OSM. If the new release does contain updates to the CRDs, it is required to first delete existing CRDs and the associated Custom Resources prior to upgrading.

In the `./scripts/cleanup` directory we have included a helper script to delete those CRDs and Custom Resources: `./scripts/crd-cleanup.sh`

After upgrading, the Custom Resources will need to be recreated using the updated CRDs installed by the upgrade.

## OSM ConfigMap
When upgrading, any edits you've made to the OSM ConfigMap may be reverted to the default. Please ensure that you carefully follow the guide to prevent these values from being overwritten.

## Upgrade OSM using Helm
Use the `helm` CLI to upgrade the OSM control plane.

### Preserve ConfigMap
To preserve any edits you've made to the ConfigMap, use the `helm --set` flag. Find the corresponding field in the [values](https://github.com/openservicemesh/osm/blob/main/charts/osm/values.yaml) file and set the desired value.

For example, to keep the `envoy_log_level` field in the ConfigMap set to `info`:

```console
$ helm upgrade <mesh name> osm --repo https://openservicemesh.github.io/osm --version <chart version> --namespace <osm namespace> --reuse-values --set OpenServiceMesh.envoyLogLevel=info
```
Omit the `--set` flag if you have not edited the ConfigMap.

Run `helm upgrade --help` for more options.
