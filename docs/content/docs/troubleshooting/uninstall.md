---
title: "Uninstall Troubleshooting"
description: "OSM Mesh Uninstall Troubleshooting Guide"
type: docs
---

# OSM Mesh Uninstall Troubleshooting Guide

## Unsuccessful Uninstall

If for any reason, `osm uninstall` is unsuccessful, run the [cleanup script](https://github.com/openservicemesh/osm/blob/release-v0.8/scripts/cleanup/osm-cleanup.sh) which will delete any OSM related resources.

To run the script, create a `.env` environment variable file to set the values specified at the top of the script. These values should match the values used to deploy the mesh.

In the root directory of the osm repository locally, run:

```console
./scripts/cleanup/osm-cleanup.sh
```
