# Distributed Calculator Demo

Source: https://github.com/dapr/samples/tree/master/3.distributed-calculator

# Setup Redis

## Creating a Redis Store

(See: https://github.com/dapr/docs/blob/master/howto/configure-redis/README.md)

Dapr can use any Redis instance - containerized, running on your local dev machine, or a managed cloud service. If you already have a Redis store, move on to the [Configuration](#configuration) section.

### Creating a Redis Cache in your Kubernetes Cluster using Helm

We can use [Helm](https://helm.sh/) to quickly create a Redis instance in our Kubernetes cluster. This approach requires [Installing Helm](https://github.com/helm/helm#install).

1. Install Redis into your cluster. Note that we're explicitly setting an image tag to get a version greater than 5, which is what Dapr' pub/sub functionality requires. If you're intending on using Redis as just a state store (and not for pub/sub), you do not have to set the image version.
    ```bash
    helm repo add bitnami https://charts.bitnami.com/bitnami
    helm install redis bitnami/redis
    ```

2. Run `kubectl get pods` to see the Redis containers now running in your cluster.
3. Add `redis-master:6379` as the `redisHost` in your [redis.yaml](#configuration) file. For example:
    ```yaml
        metadata:
        - name: redisHost
          value: redis-master:6379
    ```
4. Next, we'll get our Redis password, which is slightly different depending on the OS we're using:
    - **Windows**: Run `kubectl get secret --namespace default redis -o jsonpath="{.data.redis-password}" > encoded.b64`, which will create a file with your encoded password. Next, run `certutil -decode encoded.b64 password.txt`, which will put your redis password in a text file called `password.txt`. Copy the password and delete the two files.

    - **Linux/MacOS**: Run `kubectl get secret --namespace default redis -o jsonpath="{.data.redis-password}" | base64 --decode` and copy the outputted password.

    Add this password as the `redisPassword` value in your [redis.yaml](#configuration) file. For example:
    ```yaml
        metadata:
        - name: redisPassword
          value: lhDOkwTlp0
    ```


## Apply the configuration

### Kubernetes

```
kubectl apply -f redis.yaml
```

### Standalone

To run locally, create a `components` dir containing the YAML file and provide the path to the `dapr run` command with the flag `--components-path`.
