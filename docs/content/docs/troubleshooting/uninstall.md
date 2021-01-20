---
title: "Uninstall Troubleshooting"
description: "OSM Uninstall Troubleshooting Guide"
type: docs
aliases: ["troubleshooting"]
---

# OSM Uninstall Troubleshooting Guide

## Leaked Resources
If the [uninstallation guide](../../uninstallation_guide) was not followed, it is possible that resources could be leaked.

If the Helm release, OSM controller, or their respective namespaces are deleted, then the `osm` CLI won't be able to uninstall any remaining resources, particularly if they are cluster scoped.

These leaked resources result in an error when trying to install a new mesh with the same name but different namespace. 

```
Error: rendered manifests contain a resource that already exists. Unable to continue with install: ClusterRole "osm" in namespace "" exists and cannot be imported into the current release: invalid ownership metadata; annotation validation error: key "meta.helm.sh/release-namespace" must equal "osm-system2": current value is "osm-system"
```

In the `./scripts/cleanup` directory we have included a helper script to delete those leaked resources: `./scripts/cleanup/osm-cleanup.sh`

To run the script, create a `.env` environment variable file to set the values specified at the top of the script. These values should match the values used to deploy the mesh.
