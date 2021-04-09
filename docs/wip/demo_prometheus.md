---
title: "Prometheus Demo"
description: "A simple demo showing how OSM integrates with Prometheus for metrics"
type: docs
---

To get a taste of how OSM works with Prometheus, try installing a new mesh with sample applications to see which metrics are collected.

1. Install OSM with its own Prometheus instance:

    ```console
    $ osm install --deploy-prometheus --enable-permissive-traffic-policy
    OSM installed successfully in namespace [osm-system] with mesh name [osm]
    ```

1. Create a namespace for sample workloads:

    ```console
    $ kubectl create namespace metrics-demo
    namespace/metrics-demo created
    ```

1. Make the new OSM monitor the new namespace:

    ```console
    $ osm namespace add metrics-demo
    Namespace [metrics-demo] successfully added to mesh [osm]
    ```

1. Configure OSM's Prometheus to scrape metrics from the new namespace:

    ```console
    $ osm metrics enable --namespace metrics-demo
    Metrics successfully enabled in namespace [metrics-demo]
    ```

1. Install sample applications:

    ```console
    $ kubectl apply -f docs/example/manifests/samples/curl/curl.yaml -n metrics-demo
    serviceaccount/curl created
    deployment.apps/curl created
    $ kubectl apply -f docs/example/manifests/samples/httpbin/httpbin.yaml -n metrics-demo
    serviceaccount/httpbin created
    service/httpbin created
    deployment.apps/httpbin created
    ```

    Ensure the new Pods are Running and all containers are ready:

    ```console
    $ kubectl get pods -n metrics-demo
    NAME                       READY   STATUS    RESTARTS   AGE
    curl-54ccc6954c-q8s89      2/2     Running   0          95s
    httpbin-8484bfdd46-vq98x   2/2     Running   0          72s
    ```

1. Generate traffic:

    The following command makes the curl Pod make about 1 request per second to the httpbin Pod forever:

    ```console
    $ kubectl exec -n metrics-demo -ti "$(kubectl get pod -n metrics-demo -l app=curl -o jsonpath='{.items[0].metadata.name}')" -c curl -- sh -c 'while :; do curl -i httpbin.metrics-demo:14001/status/200; sleep 1; done'
    HTTP/1.1 200 OK
    server: envoy
    date: Tue, 23 Mar 2021 17:27:44 GMT
    content-type: text/html; charset=utf-8
    access-control-allow-origin: *
    access-control-allow-credentials: true
    content-length: 0
    x-envoy-upstream-service-time: 1

    HTTP/1.1 200 OK
    server: envoy
    date: Tue, 23 Mar 2021 17:27:45 GMT
    content-type: text/html; charset=utf-8
    access-control-allow-origin: *
    access-control-allow-credentials: true
    content-length: 0
    x-envoy-upstream-service-time: 2

    ...
    ```

1. View metrics in Prometheus:

    Forward the Prometheus port:

    ```console
    $ kubectl port-forward -n osm-system $(kubectl get pods -n osm-system -l app=osm-prometheus -o jsonpath='{.items[0].metadata.name}') 7070
    Forwarding from 127.0.0.1:7070 -> 7070
    Forwarding from [::1]:7070 -> 7070
    ```

    Navigate to http://localhost:7070 in a web browser to view the Prometheus UI. The following query shows how many requests per second are being made from the curl pod to the httpbin pod, which should be about 1:

    ```
    irate(envoy_cluster_upstream_rq_xx{source_service="curl", envoy_cluster_name="metrics-demo/httpbin"}[30s])
    ```

    Feel free to explore the other metrics available from within the Prometheus UI.

1. Cleanup

    Once you are done with the demo resources, clean them up by first deleting the application namespace:

    ```console
    $ kubectl delete ns metrics-demo
    namespace "metrics-demo" deleted
    ```

    Then, uninstall OSM:

    ```
    $ osm uninstall
    Uninstall OSM [mesh name: osm] ? [y/n]: y
    OSM [mesh name: osm] uninstalled
    ```
