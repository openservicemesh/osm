---
title: "Install Troubleshooting"
description: "OSM Mesh Install Troubleshooting Guide"
type: docs
---

# OSM Mesh Install Troubleshooting Guide

## Leaked Resources

During an improper or incomplete uninstallation, it is possible that OSM resources could be left behind in a Kubernetes cluster.

For example, if the Helm release, OSM controller, or their respective namespaces are deleted, then the `osm` CLI won't be able to uninstall any remaining resources, particularly if they are cluster scoped.

As a result, one may see this error during a subsequent install of a new mesh with the same name but different namespace:

```console
Error: rendered manifests contain a resource that already exists. Unable to continue with install: ClusterRole "<mesh-name>" in namespace "" exists and cannot be imported into the current release: invalid ownership metadata; annotation validation error: key "meta.helm.sh/release-namespace" must equal "<new-namespace>": current value is "<old-namespace>"
```

In the case of this error, use the [cleanup script](https://github.com/openservicemesh/osm/blob/release-v0.8/scripts/cleanup/osm-cleanup.sh) located in the osm repository to delete any remaining resources.

To run the script, create a `.env` environment variable file to set the values specified at the top of the script. These values should match the values used to deploy the mesh.

In the root directory of the osm repository locally, run:

```console
./scripts/cleanup/osm-cleanup.sh
```

Then, try installing OSM again on the cluster.
