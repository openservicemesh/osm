# Uninstallation Guide

This guide describes how to uninstall Open Service Mesh (OSM) from a Kubernetes cluster using the `osm` CLI.

## Prerequisites
- Kubernetes cluster with OSM installed
- The [osm CLI](installation_guide.md#Set-up-the-OSM-CLI)

## Uninstall OSM
Use the `osm` CLI to uninstall the OSM control plane from a Kubernetes cluster.

Run `osm mesh uninstall`. 
```console
# Uninstall osm control plane components
$ osm mesh uninstall
Uninstall OSM [mesh name: osm] ? [y/n]: y
OSM [mesh name: osm] uninstalled
```

Run `osm mesh uninstall --help` for more options.