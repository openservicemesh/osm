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

In the `./scripts/cleanup` directory we have included a helper script to delete those CRDs and Custom Resources: `./scripts/cleanup/crd-cleanup.sh`

After upgrading, the CRDs and Custom Resources will need to be recreated.

1. Checkout the correct release branch of the [repo](https://github.com/openservicemesh/osm).
1. Install the new CRDs. (Run from the root of the repo.)
    - `kubectl apply -f charts/osm/crds/`
1. Recreate CustomResources

## OSM Configuration
When upgrading, any custom settings used to install or run OSM may be reverted to the default. This includes any metrics deployments and any changes to the OSM ConfigMap. Please ensure that you carefully follow the guide to prevent these values from being overwritten.

## Upgrade OSM using Helm
Use the `helm` CLI to upgrade the OSM control plane.

### Preserve OSM configuration
To preserve any changes you've made to the OSM configuration, use the `helm --values` flag. Create a copy of the [values file](https://github.com/openservicemesh/osm/blob/main/charts/osm/values.yaml) (make sure to use the version for the upgraded chart) and change any values you wish to customize. You can omit all other values.

To see which values correspond to the ConfigMap settings, see the [OSM ConfigMap documentation](osm_config_map.md)

For example, to keep the `envoy_log_level` field in the ConfigMap set to `info`, save the following as `override.yaml`:

```
OpenServiceMesh:
  envoyLogLevel: info
```
<b>Warning:</b> Do NOT change `OpenServiceMesh.meshName` or `OpenServiceMesh.osmNamespace`

### Helm Upgrade
Then run the following `helm upgrade` command.
```console
$ helm upgrade <mesh name> osm --repo https://openservicemesh.github.io/osm --version <chart version> --namespace <osm namespace> --values override.yaml
```
Omit the `--values` flag if you prefer to use the default settings, but please note this could override any edits you've made to the ConfigMap.

Run `helm upgrade --help` for more options.
