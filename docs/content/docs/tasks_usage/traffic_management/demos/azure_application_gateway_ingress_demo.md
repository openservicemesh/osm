---
title: "Ingress with Azure Application Gateway Demo"
description: "Exposing services outside the cluster using Azure Application Gateway Ingress Controller"
type: docs
aliases: ["azure_application_gateway_ingress_demo.md"]
---

The OSM ingress guide with Azure Application Gateway is a short demo on exposing HTTP routes on services within the mesh externally using the Azure Application Gateway ingress controller. 

## Sample demo

### HTTP traffic with ingress

The following demo sends a request from an external IP to a httpbin service inside the cluster.

1. Create an Azure Kubernetes Service (AKS) cluster with [Application Gateway](https://azure.github.io/application-gateway-kubernetes-ingress/tutorials/tutorial.general/) and install the application gateway ingress controller on the cluster.

    Verify that the ingress-azure pod is up and running in the default namespace:

    ```console
    $ kubectl get pods
    NAME                             READY   STATUS    RESTARTS   AGE
    ingress-azure-5cdf9b7586-z66m9   1/1     Running   0          96m
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
    httpbin-74677b7df7-zzlm2   2/2     Running   0          11h
    ```

    ```console
    $ kubectl get svc -n httpbin
    NAME      TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)     AGE
    httpbin   ClusterIP   10.0.22.196   <none>        14001/TCP   11h
    ```

1. Apply an ingress configuration yaml to expose the HTTP path `/status/200` on the `httpbin` service with `kubectl apply -f`
    ```yaml
    apiVersion: networking.k8s.io/v1beta1
    kind: Ingress
    metadata:
      name: httpbin-ingress
      namespace: httpbin
      annotations:
        kubernetes.io/ingress.class: azure/application-gateway
    spec:
      rules:
      - http:
          paths:
          - path: /status/200
            backend:
              serviceName: httpbin
              servicePort: 14001
    ```

    Confirm that the httpbin-ingress has been successfully deployed.

    ```console
    $ kubectl get ingress -n httpbin
    NAMESPACE   NAME              CLASS    HOSTS   ADDRESS        PORTS   AGE
    httpbin     httpbin-ingress   <none>   *       20.69.68.127   80      7m18s
    ```

1. Confirm that a request to the httpbin service from the external IP address of the Ingress resource succeeds (in this case, the external address would be `20.69.68.127`)

    ```bash
    curl http://<external-ip>/status/200
    ```
