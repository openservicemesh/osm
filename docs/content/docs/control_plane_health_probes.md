---
title: "Health Probes: OSM Control Plane"
description: "How OSM's health probes work and what to do if they fail"
type: "docs"
---

# OSM Control Plane Health Probes

OSM control plane components leverage health probes to communicate their overall status. Health probes are implemented as HTTP endpoints which respond to requests with HTTP status codes indicating success or failure.

Kubernetes uses these probes to communicate the status of the control plane Pods' statuses and perform some actions automatically to improve availability. More details about Kubernetes probes can be found [here](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/).

## OSM Components with Probes

The following OSM control plane components have health probes:

#### osm-controller

The following HTTP endpoints are available on osm-controller on port 9091:

- `/health/alive`: HTTP 200 response code indicates OSM's Aggregated Discovery Service (ADS) is running. No response is sent when the service is not yet running.

- `/health/ready`: HTTP 200 response code indicates ADS is ready to accept gRPC connections from proxies. HTTP 503 or no response indicates gRPC connections from proxies will not be successful.

#### osm-injector

The following HTTP endpoints are available on osm-injector on port 9090:

- `/healthz`: HTTP 200 response code indicates the injector is ready to inject new Pods with proxy sidecar containers. No response is sent otherwise.

## How to Verify OSM Health

Because OSM's Kubernetes resources are configured with liveness and readiness probes, Kubernetes will automatically poll the health endpoints on the osm-controller and osm-injector Pods.

When a liveness probe fails, Kubernetes will generate an Event (visible by `kubectl describe pod <pod name>`) and restart the Pod. The `kubectl describe` output may look like this:

```
...
Events:
  Type     Reason     Age               From               Message
  ----     ------     ----              ----               -------
  Normal   Scheduled  24s               default-scheduler  Successfully assigned osm-system/osm-controller-85fcb445b-fpv8l to osm-control-plane
  Normal   Pulling    23s               kubelet            Pulling image "openservicemesh/osm-controller:v0.8.0"
  Normal   Pulled     23s               kubelet            Successfully pulled image "openservicemesh/osm-controller:v0.8.0" in 562.2444ms
  Normal   Created    1s (x2 over 23s)  kubelet            Created container osm-controller
  Normal   Started    1s (x2 over 23s)  kubelet            Started container osm-controller
  Warning  Unhealthy  1s (x3 over 21s)  kubelet            Liveness probe failed: HTTP probe failed with statuscode: 503
  Normal   Killing    1s                kubelet            Container osm-controller failed liveness probe, will be restarted
```

When a readiness probe fails, Kubernetes will generate an Event (visible with `kubectl describe pod <pod name>`) and ensure no traffic destined for Services the Pod may be backing is routed to the unhealthy Pod. The `kubectl describe` output for a Pod with a failing readiness probe may look like this:

```
...
Events:
  Type     Reason     Age               From               Message
  ----     ------     ----              ----               -------
  Normal   Scheduled  36s               default-scheduler  Successfully assigned osm-system/osm-controller-5494bcffb6-tn5jv to osm-control-plane
  Normal   Pulling    36s               kubelet            Pulling image "openservicemesh/osm-controller:latest"
  Normal   Pulled     35s               kubelet            Successfully pulled image "openservicemesh/osm-controller:v0.8.0" in 746.4323ms
  Normal   Created    35s               kubelet            Created container osm-controller
  Normal   Started    35s               kubelet            Started container osm-controller
  Warning  Unhealthy  4s (x3 over 24s)  kubelet            Readiness probe failed: HTTP probe failed with statuscode: 503
```

The Pod's `status` will also indicate that it is not ready which is shown in its `kubectl get pod` output. For example:

```
NAME                              READY   STATUS    RESTARTS   AGE
osm-controller-5494bcffb6-tn5jv   0/1     Running   0          26s
```

The Pods' health probes may also be invoked manually by forwarding the Pod's necessary port and using `curl` or any other HTTP client to issue requests. For example, to verify the liveness probe for osm-controller, get the Pod's name and forward port 9091:

```
# Assuming OSM is installed in the osm-system namespace
kubectl port-forward -n osm-system $(kubectl get pods -n osm-system -l app=osm-controller -o jsonpath='{.items[0].metadata.name}') 9091
```

Then, in a separate terminal instance, `curl` may be used to check the endpoint. The following example shows a healthy osm-controller:

```console
$ curl -i localhost:9091/health/alive
HTTP/1.1 200 OK
Date: Thu, 18 Mar 2021 20:15:29 GMT
Content-Length: 16
Content-Type: text/plain; charset=utf-8

Service is alive
```

## Troubleshooting

If any health probes are consistently failing, perform the following steps to identify the root cause:

1. Ensure the unhealthy osm-controller or osm-injector Pod is not running an Envoy sidecar container.

    To verify The osm-controller Pod is not running an Envoy sidecar container, verify none of the Pod's containers' images is an Envoy image. Envoy images have "envoyproxy/envoy" in their name.

    For example, an osm-controller Pod that includes an Envoy container:
    ```console
    $ # Assuming OSM is installed in the osm-system namespace:
    $ kubectl get pod -n osm-system $(kubectl get pods -n osm-system -l app=osm-controller -o jsonpath='{.items[0].metadata.name}') -o jsonpath='{range .spec.containers[*]}{.image}{"\n"}{end}'
    openservicemesh/osm-controller:v0.8.0
    envoyproxy/envoy-alpine:v1.17.1
    ```

    To verify The osm-injector Pod is not running an Envoy sidecar container, verify none of the Pod's containers' images is an Envoy image. Envoy images have "envoyproxy/envoy" in their name.

    For example, an osm-injector Pod that includes an Envoy container:
    ```console
    $ # Assuming OSM is installed in the osm-system namespace:
    $ kubectl get pod -n osm-system $(kubectl get pods -n osm-system -l app=osm-injector -o jsonpath='{.items[0].metadata.name}') -o jsonpath='{range .spec.containers[*]}{.image}{"\n"}{end}'
    openservicemesh/osm-injector:v0.8.0
    envoyproxy/envoy-alpine:v1.17.1
    ```

    If either Pod is running an Envoy container, it may have been injected erroneously by this or another another instance of OSM. For each mesh found with the `osm mesh list` command, verify the OSM namespace of the unhealthy Pod is not listed in the `osm namespace list` output with `SIDECAR-INJECTION` "enabled" for any OSM instance found with the `osm mesh list` command.

    For example, for all of the following meshes:

    ```console
    $ osm mesh list
    
    MESH NAME   NAMESPACE      CONTROLLER PODS                   VERSION
    osm         osm-system     osm-controller-5494bcffb6-qpjdv   v0.8.2
    osm2        osm-system-2   osm-controller-48fd3c810d-sornc   v0.8.2
    ```

    Note how `osm-system` is present in the following list:

    ```console
    $ osm namespace list --mesh-name osm --osm-namespace osm-system
    NAMESPACE    MESH    SIDECAR-INJECTION
    osm-system   osm2    enabled
    bookbuyer    osm2    enabled
    bookstore    osm2    enabled
    ```

    If the OSM namespace is found in any `osm namespace list` command with `SIDECAR-INJECTION` enabled, remove the namespace from the mesh injecting the sidecars. For the example above:

    ```console
    $ osm namespace remove osm-system --mesh-name osm2 --osm-namespace osm-system2
    ```

1. Determine if Kubernetes encountered any errors while scheduling or starting the Pod.

    Look for any errors that may have recently occurred with `kubectl describe` of the unhealthy Pod.

    For osm-controller:

    ```console
    $ # Assuming OSM is installed in the osm-system namespace:
    $ kubectl describe pod -n osm-system $(kubectl get pods -n osm-system -l app=osm-controller -o jsonpath='{.items[0].metadata.name}')
    ```

    For osm-injector:

    ```console
    $ # Assuming OSM is installed in the osm-system namespace:
    $ kubectl describe pod -n osm-system $(kubectl get pods -n osm-system -l app=osm-injector -o jsonpath='{.items[0].metadata.name}')
    ```

    Resolve any errors and verify OSM's health again.

1. Determine if the Pod encountered a runtime error.

    Look for any errors that may have occurred after the container started by inspecting its logs. Specifically, look for any logs containing the string `"level":"error"`.

    For osm-controller:

    ```console
    $ # Assuming OSM is installed in the osm-system namespace:
    $ kubectl logs -n osm-system $(kubectl get pods -n osm-system -l app=osm-controller -o jsonpath='{.items[0].metadata.name}')
    ```

    For osm-injector:

    ```console
    $ # Assuming OSM is installed in the osm-system namespace:
    $ kubectl logs -n osm-system $(kubectl get pods -n osm-system -l app=osm-injector -o jsonpath='{.items[0].metadata.name}')
    ```

    Resolve any errors and verify OSM's health again.
