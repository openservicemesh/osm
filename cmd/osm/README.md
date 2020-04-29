# osm install instructions for testing


## Setting up your environment
_Note: Ensure [prerequisites](https://github.com/open-service-mesh/osm/blob/master/DEMO.md#prerequisites) are met._

Currently, you must build and push all the `osm` image artifacts to your own Azure container registry to install and test the service mesh. Build and push all osm related image artifacts using the script:
```console
$ ./demo/build-push-images.sh
```

## Install osm control plane
Build the `osm` binary:
```console
$ make build-osm
```

Create a container registry secret to be able to pull OSM images from your registry
```console
$ bin/osm config acr-secret [your-registry.azurecr.io]
```

Install the osm control plane
```console
$ bin/osm install --container-registry <your-registry>.azurecr.io
$ kubectl get pods -n osm-system  # check if `ads-<some-hash>` pod is running
```

`osm install` will do the following:
1. Ensure the `osm-system` Kubernetes namespace exists. This is where all the control plane components will be installed.
2. Create a Kubernetes docker-registry secret called `acr-creds` for your container registry. _Note: This will only work if you are already logged into acr via `az acr login --name <your-registry>`_
3. Installs osm control plane (ads deployment, service, rbac), webhook for sidecar proxy injection, relevant certificates for both the control plane and the webhook.


## Deploy demo application
```console
$ ./demo/deploy-apps.sh # will deploy a bookbuyer, bookstore, and a booktheif
$ ./demo/tail-bookbuyer.sh # outputs logs of bookbuyer application
```
The `bookbuyer` application is making HTTP requests to the `bookstore` application to buy books. The requests from bookbuyer to bookstore are expected to succeed. You should see `200 OK` response codes in the logs. You're seeing this succeed because an [SMI Traffic Policy](https://github.com/servicemeshinterface/smi-spec/blob/master/traffic-access-control.md) has been configured to allow HTTP requests from `bookbuyer` to `bookstore.


To delete this test environment:
```console
$ ./demo/clean-kubernetes.sh
```
