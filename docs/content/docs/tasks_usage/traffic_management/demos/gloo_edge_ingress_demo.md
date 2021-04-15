---
title: "Ingress with Gloo Edge Ingress Demo"
description: "Exposing services outside the cluster using Gloo Edge Ingress Controller"
type: docs
aliases: ["gloo_edge_ingress_demo.md"]
---

The OSM ingress guide with Gloo API Gateway is a short demo on exposing HTTP routes on services within the mesh externally using the Gloo Edge ingress controller. 

## Sample demo

### HTTP traffic with ingress

The following demo sends a request from an external IP to a httpbin service inside the cluster.

1. Install the [Gloo Edge](https://docs.solo.io/gloo-edge/latest/) ingress controller with the [glooctl CLI](https://docs.solo.io/gloo-edge/latest/installation/preparation/#install-command-line-tool-cli)
    ```bash
    glooctl install ingress
    ```

    Verify that the pods in the gloo-system namespace is up and running: 

    ```console
    $ kubectl get pods -n gloo-system
    NAME                             READY   STATUS    RESTARTS   AGE
    discovery-b7c89f698-7xpw5        1/1     Running   0          173m
    gloo-7b844d6cd4-djnlk            1/1     Running   0          173m
    ingress-7ffcc9df95-7fb6j         1/1     Running   0          173m
    ingress-proxy-76cf8c6bdb-m727w   1/1     Running   0          173m
    ```

1. Install OSM onto the cluster.
    ```bash
    osm install
    ```

1. Deploy the `httpbin` service into the `httpbin` namespace after enrolling its namespace to the mesh. The `httpbin` service runs on port `14001`.
    ```bash
    # Create the httpbin namespace
    kubectl create namespace httpbin

    # Add the namespace to the mesh
    osm namespace add httpbin

    # Deploy httpbin service in the httpbin namespace
    kubectl apply -f docs/example/manifests/samples/httpbin/httpbin.yaml -n httpbin
    ```

    Confirm the `httpbin` service and pods are up and running.

    ```console
    $ kubectl get pods -n httpbin
    NAME                       READY   STATUS    RESTARTS   AGE
    httpbin-74677b7df7-7f5v4   1/1     Running   0          149m
    ```

    ```console
    $ kubectl get svc -n httpbin
    NAME      TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)     AGE
    httpbin   ClusterIP   10.0.115.107   <none>        14001/TCP   149m
    ```


1. Apply an ingress configuration yaml to expose the HTTP path `/status/200` on the `httpbin` service with `kubectl apply -f`

    > Note: Use the appropriate ingress resource based on the desired API version.

   Ingress v1 resource:
    ```yaml
    apiVersion: networking.k8s.io/v1
    kind: Ingress
    metadata:
      name: httpbin-ingress
      namespace: httpbin
    spec:
      ingressClassName: gloo
      rules:
      - http:
          paths:
          - path: /status/200
            pathType: ImplementationSpecific # Must be one of: Exact, Prefix, ImplementationSpecific
            backend:
              service:
                name: httpbin
                port:
                  number: 14001
    ```

    Ingress v1beta1 resource:
    ```yaml
    apiVersion: networking.k8s.io/v1beta1
    kind: Ingress
    metadata:
      name: httpbin-ingress
      namespace: httpbin
      annotations:
        kubernetes.io/ingress.class: gloo
    spec:
      rules:
      - http:
          paths:
          - path: /status/200
            pathType: ImplementationSpecific # Must be one of: Exact, Prefix, ImplementationSpecific
            backend:
              serviceName: httpbin
              servicePort: 14001
    ```


1. Confirm that the httpbin-ingress has been successfully deployed.

    ```console
    $ kubectl get ingress -n httpbin
    NAME              CLASS    HOSTS                               ADDRESS         PORTS   AGE
    httpbin-ingress   <none>   httpbin.httpbin.svc.cluster.local   52.234.160.38   80      152m
    ```

1. Configure the `Upstream` object to use OSM's root ca bundle: 

    ```yaml
    apiVersion: gloo.solo.io/v1
    kind: Upstream
    metadata:
      name: httpbin-httpbin-14001
      namespace: gloo-system
    spec:
      sslConfig:
        sni: "httpbin.httpbin.svc.cluster.local"
        secretRef:
          name: osm-ca-bundle
          namespace: gloo-system
    kube:
      selector:
        app: httpbin
      serviceName: httpbin
      serviceNamespace: httpbin
      servicePort: 14001
    ```

1. Confirm that a request to the httpbin service via the external IP address of the Ingress resource succeeds (in this case, the external address would be `52.234.160.38`)

    ```bash
    curl http://<external-ip>/status/200
    ```

At this point you can call your Ingress endpoint and get HTTPS traffic from the edge to your OSM service. As a convenience, you can run the following to get your ingress hostname/IP:

```bash
glooctl proxy url --name ingress-proxy
```