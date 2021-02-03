---
title: "Upgrade Guide"
description: "Upgrade Guide"
type: docs
---

# Upgrade Guide

This guide describes how to upgrade the Open Service Mesh (OSM) control plane.

## How upgrades work

OSM's control plane lifecycle is managed by Helm and can be upgraded with [Helm's upgrade functionality](https://helm.sh/docs/intro/using_helm/#helm-upgrade-and-helm-rollback-upgrading-a-release-and-recovering-on-failure), which will patch or replace control plane components as needed based on changed values and resource templates.

### Resource availability during upgrade
Since upgrades may include redeploying the osm-controller with the new version, there may be some downtime of the controller. While the osm-controller is unavailable, there will be a delay in processing new SMI resources, creating new pods to be injected with a proxy sidecar container will fail, and mTLS certificates will not be rotated.

However, already existing SMI resources will be unaffected (assuming [CRD Upgrades](#CRD-Upgrades) are not needed). This means that the data plane (which includes the Envoy sidecar configs) will also be unaffected by upgrading.

Data plane interruptions are expected if the upgrade includes CRD changes. Streamlining data plane upgrades is being tracked in issue [#512](https://github.com/openservicemesh/osm/issues/512).

## Policy

Only certain upgrade paths are tested and supported.

**Note**: These plans are tentative and subject to change.

Breaking changes in this section refer to incompatible changes to the following user-facing components:
- `osm` CLI commands, flags, and behavior
- SMI CRDs and controllers

This implies the following are NOT user-facing and incompatible changes are NOT considered "breaking" as long as the incompatibility is handled by user-facing components:
- Chart values.yaml
- `osm-config` ConfigMap
- Internally-used labels and annotations (monitored-by, injection, metrics, etc.)

Upgrades are only supported between versions that do not include breaking changes, as described below.

For OSM versions `0.y.z`:
- Breaking changes will not be introduced between `0.y.z` and `0.y.z+1`
- Breaking changes may be introduced between `0.y.z` and `0.y+1.0`

For OSM versions `x.y.z` where `x >= 1`:
- Breaking changes will not be introduced between `x.y.z` and `x.y+1.0` or between `x.y.z` and `x.y.z+1`
- Breaking changes may be introduced between `x.y.z` and `x+1.0.0`

## How to upgrade OSM

The recommended way to upgrade a mesh is with the `osm` CLI. For advanced use cases, `helm` may be used.

### CRD Upgrades
Because Helm does not manage CRDs beyond the initial installation, special care needs to be taken during upgrades when CRDs are changed. Please check the `CRD Updates` section of the [release notes](https://github.com/openservicemesh/osm/releases) to see if additional steps are required to update the CRDs used by OSM. If the new release does contain updates to the CRDs, it is required to first delete existing CRDs and the associated Custom Resources prior to upgrading.

In the `./scripts/cleanup` directory we have included a helper script to delete those CRDs and Custom Resources: `./scripts/cleanup/crd-cleanup.sh`

After upgrading, the CRDs and Custom Resources will need to be recreated.

1. Checkout the tag of the [repo](https://github.com/openservicemesh/osm) corresponding to the version of the upgraded chart.
1. Install the new CRDs. (Run from the root of the repo.)
    - `kubectl apply -f charts/osm/crds/`
1. Recreate CustomResources

Improving CRD upgrades is being tracked in [#893](https://github.com/openservicemesh/osm/issues/893).

### Upgrading with the OSM CLI

**Pre-requisites**

- Kubernetes cluster with the OSM control plane installed
- `osm` CLI installed
  - By default, the `osm` CLI will upgrade to the same chart version that it installs. e.g. v0.8.0 of the `osm` CLI will upgrade to v0.8.0 of the OSM Helm chart.

The `osm mesh upgrade` command performs a `helm upgrade` of the existing Helm release for a mesh.

Basic usage requires no additional arguments or flags:
```console
$ osm mesh upgrade
OSM successfully upgraded mesh osm
```

This command will upgrade the mesh with the default mesh name in the default OSM namespace. Values from the previous release will carry over to the new release except for `OpenServiceMesh.image.registry` and `OpenServiceMesh.image.tag` which are overridden by default. For example, if OSM v0.7.0 is installed, `osm mesh upgrade` for v0.8.0 of the CLI will update the control plane images to v0.8.0 by default.

See `osm mesh upgrade --help` for more details

### Upgrading with Helm

#### Pre-requisites

- Kubernetes cluster with the OSM control plane installed
- The [helm 3 CLI](https://helm.sh/docs/intro/install/) 

#### OSM Configuration
When upgrading, any custom settings used to install or run OSM may be reverted to the default. This includes any metrics deployments and any changes to the OSM ConfigMap. Please ensure that you carefully follow the guide to prevent these values from being overwritten.

To preserve any changes you've made to the OSM configuration, use the `helm --values` flag. Create a copy of the [values file](https://github.com/openservicemesh/osm/blob/main/charts/osm/values.yaml) (make sure to use the version for the upgraded chart) and change any values you wish to customize. You can omit all other values.

To see which values correspond to the ConfigMap settings, see the [OSM ConfigMap documentation](osm_config_map.md)

For example, to keep the `envoy_log_level` field in the ConfigMap set to `info`, save the following as `override.yaml`:

```
OpenServiceMesh:
  envoyLogLevel: info
```
<b>Warning:</b> Do NOT change `OpenServiceMesh.meshName` or `OpenServiceMesh.osmNamespace`

#### Helm Upgrade
Then run the following `helm upgrade` command.
```console
$ helm upgrade <mesh name> osm --repo https://openservicemesh.github.io/osm --version <chart version> --namespace <osm namespace> --values override.yaml
```
Omit the `--values` flag if you prefer to use the default settings, but please note this could override any edits you've made to the ConfigMap.

Run `helm upgrade --help` for more options.
