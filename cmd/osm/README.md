# osm install instructions for testing

Note: Ensure [prerequisites](https://github.com/open-service-mesh/osm/blob/master/DEMO.md#prerequisites) are met and environment variables are set for these scripts to work.

```console
$ ./demo/build-push-images.sh
$ ./demo/clean-kubernetes.sh
$ ./demo/create-namespaces.sh

$ ./demo/create-container-registry-creds.sh // creates k8s secret for container registry creds

$ make build-cert
$ make build-osm
$ bin/osm install --container-registry <your-acr-registry.azurecr.io>
$ kubectl get pods -n osm-system

$ ./demo/deploy-apps.sh

```

To delete this test environment:
```console
$ ./demo/clean-kubernetes.sh
```
