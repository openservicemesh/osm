# osm install instructions for testing

Note: Ensure [prerequisites](https://github.com/open-service-mesh/osm/blob/master/DEMO.md#prerequisites) are met and environment variables are set for these scripts to work.

```console
$ ./demo/build-push-images.sh
$ ./demo/clean-kubernetes.sh

$ ./demo/create-container-registry-creds.sh // creates k8s secret for container registry creds

$ make build-osm
$ bin/osm install --container-registry <your-acr-registry.azurecr.io>
$ k get pods -n osm
```

To delete this test environment:
```console
$ ./demo/clean-kubernetes.sh
```
