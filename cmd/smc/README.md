# smc install instructions for testing

Note: Ensure [prerequisites](https://github.com/deislabs/smc/blob/master/DEMO.md#prerequisites) are met and environment variables are set for these scripts to work.

```console
$ ./demo/build-push-images.sh
$ kubectl create namespace smc
$ ./demo/create-container-registry-creds.sh // creates k8s secret

$ make build-smc
$ bin/smc install --container-registry <your-acr-registry.azurecr.io>
$ k get pods -n smc
```

To delete this test environment:
```console
$ ./demo/clean-kubernetes.sh
$ kubectl delete clusterrole smc-xds
$ kubectl delete clusterrolebinding smc-xds
```
