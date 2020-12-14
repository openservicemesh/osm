# OSM Uninstall Troubleshooting Guide

## Leaked Resources
If the Helm release, OSM controller, or their respective namespaces are deleted, then it is possible that the `helm ` CLI and `osm` CLI won't be able to uninstall any remaining resources, particularly if they are cluster scoped.

These leaked resources result in an error when trying to install a new mesh with the same name but different namespace. 

```
Error: rendered manifests contain a resource that already exists. Unable to continue with install: ClusterRole "osm" in namespace "" exists and cannot be imported into the current release: invalid ownership metadata; annotation validation error: key "meta.helm.sh/release-namespace" must equal "osm-system2": current value is "osm-system"
```

In the `./scripts` directory we have included a helper script to delete those leaked resources: `./scripts/osm-cleanup.sh`

To run the script, create a `.env` environment variable file to set the values specified at the top of the script. These values should match the values used to deploy the mesh.

After running the script, you may see some errors like this.
```
Error from server (NotFound): error when deleting "STDIN": services "osm-controller" not found
```

Those errors should be okay as long as the environment variables were set properly. They indicate that the specified resources have been deleted prior to running the script.
