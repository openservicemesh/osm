# smc install instructions for testing

Note: Ensure [prerequisites](https://github.com/deislabs/smc/blob/master/DEMO.md#prerequisites) are met and environment variables are set for these scripts to work.

```console
$ ./demo/build-push-images.sh
$ ./demo/create-certificates.sh

$ ./demo/clean-kubernetes.sh //delete smc namespace
$ ./demo/create-container-registry-creds.sh // creates k8s secret
$ ./demo/deploy-certificates-config.sh // puts certs in k8s ConfigMap
$ ./demo/deploy-envoyproxy-config.sh // creates k8s ConfigMap
$ ./demo/deploy-secrets.sh

$ kubectl create configmap kubeconfig --from-file="$HOME/.kube/config" -n smc

$ make build-smc
$ bin/smc install --container-registry <your-acr-registry.azurecr.io>
$ k get pods -n smc
```

To delete this test environment:
```console
$ ./demo/clean-kubernetes.sh
```
