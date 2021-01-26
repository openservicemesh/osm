# OSM Upgrade Troubleshooting Guide

## Server could not find requested resource
If the [upgrade CRD guide](../upgrade_guide.md##crd-upgrades) was not followed, it is possible that the installed CRDs are out of sync with the OSM controller.

The OSM controller will then crash with errors similar to this:
```
reflector.go:178] pkg/mod/k8s.io/client-go@v0.18.6/tools/cache/reflector.go:125: Failed to list *v1alpha2.TrafficTarget: the server could not find the requested resource (get traffictargets.access.smi-spec.io)
```
To resolve these errors:
1. Checkout the correct release branch of the [repo](https://github.com/openservicemesh/osm) and run the following commands from the root. 
1. Delete existing CRDs and Custom Resources (TrafficTargets, TrafficSplits, etc.)
   - `./scripts/cleanup/crd-cleanup.sh`
1. Install the new CRDs
   - `kubectl apply -f charts/osm/crds/`
1. Restart the osm-controller pod
1. Recreate CustomResources 
