---
title: "Ingress"
description: "Exposing services outside the cluster using Kubernetes Ingress"
type: docs
aliases: ["ingress.md"]
---

# Exposing services outside the cluster using Kubernetes Ingress

The OSM ingress guide is a walkthrough on exposing HTTP and HTTPS routes on services within the mesh externally using the Kubernetes Ingress API.

## Prerequisites
- An instance of OSM must be running in the cluster.
- The service needing to be exposed using Ingress needs to belong to a namespace monitored by OSM. Refer to the [Readme][1] for details.
- The ingress resource must belong to the same namespace as the backend service.
- A sidecar must be injected to the pod hosting the service, either using automatic sidecar injection or by manually annotating the pod spec for sidecar injection. Refer to the [Readme][1] for details.
- An ingress controller must be running in the cluster.

## What is ingress?

Ingress refers to providing external access to services inside the cluster, typically HTTP/HTTPS services. In kubernetes, Ingress consists of an Ingress API resource and an Ingress controller. The Kubernetes Ingress API manifest consists of declarations of how external clients are routed to a service inside the cluster, and the Ingress controller executes the routing declarations specified in the manifest.

## Why do you need ingress on your cluster?

Initially, when you group pods under a service, this service is only accessible inside of the cluster. One option, of course, is to use a load balancer. However, the downside of this approach is that each service would require a new hosted load balancer, meaning more consumption of cloud resources.

Ingress allows you to easily establish rules for traffic routing without creating several load balancers. The Ingress Controller is built using reverse proxies, allowing it to act as a load balancer. Though the Ingress Controller requires a service to expose them to services outside of the cluster, this will be the only entrypoint that outside services need to access all the internal services on your cluster, as the Ingress Controller will redirect this traffic to the internal pods as specified in the Ingress Manifest.

## What types of ingress are supported by OSM v0.8.0?

Currently, OSM supports HTTP ingress. HTTPS ingress support is experimental, with support for one-way TLS authentication to backend services.

## Supported Kubernetes Ingress API versions

Since OSM only supports Kubernetes cluster versions >= v1.18.0, the API version for the Kubernetes Ingress resource must be precisely one of `networking.k8s.io/v1` or `networking.k8s.io/v1beta1`. OSM controller dynamically negotiates the Ingress API versions served by the Kubernetes API server and enables the same versions to be served in OSM.

> Note: If either of these versions are not served by the Kubernetes API server for some reason, OSM controller will exit on failing to initialize its ingress client. This likely indicates an unsupported Kubernetes version on the cluster.

## Ingress controller compatibility

Ingress in OSM has been tested with the following ingress controllers:
- [Kubernetes Nginx Ingress Controller][2]
- [Azure Application Gateway Ingress Controller][3]
- [Gloo API Gateway][5]

Other ingress controllers might also work as long as they use the Kubernetes Ingress API. In addition, ingress controllers must allow provisioning a custom root certificate for backend server certificate validation while using HTTPS ingress.

## Configuring Ingress

### Enabling HTTP or HTTPS Ingress

By default, OSM configures HTTP as the backend protocol for services when an ingress resource is applied with a backend service that belongs to the mesh. A mesh-wide configuration setting in OSM's `osm-config` ConfigMap enables configuring ingress with the backend protocol to be HTTPS.

#### HTTP ingress
HTTP based ingress is provisioned in OSM by default.

#### HTTPS ingress
HTTPs Ingress is disabled by default when OSM is installed. However, HTTPS ingress can be enabled by updating the `osm-config` ConfigMap in `osm-controller`'s namespace (`osm-system` by default).

Patch the ConfigMap by setting use_https_ingress: "true".

```bash
# Replace osm-system with osm-controller's namespace if using a non default namespace
kubectl patch ConfigMap osm-config -n osm-system -p '{"data":{"use_https_ingress":"true"}}' --type=merge
```
> Note: Changes made with `kubectl patch` are not preserved across release upgrades. To make this change persistent between upgrades, use `osm mesh upgrade`. See `osm mesh upgrade --help` for more details.

> Note: Enabling HTTPS ingress will disable HTTP ingress.


### Disabling HTTP or HTTPS Ingress

#### HTTP ingress

HTTP ingress can be disabled by enabling HTTPS ingress.

#### HTTPs ingress

Patch the ConfigMap by setting use_https_ingress: "false".

> Note: Disabling HTTPS ingress will enable HTTP ingress.

```bash
# Replace osm-system with osm-controller's namespace if using a non default namespace
kubectl patch ConfigMap osm-config -n osm-system -p '{"data":{"use_https_ingress":"false"}}' --type=merge
```

## How it works

### Exposing an HTTP or HTTPS service using Ingress
A service can expose HTTP or HTTPS routes outside the cluster using Kubernetes Ingress along with an ingress controller. Once an ingress resource is configured to expose HTTP routes outside the cluster to a service within the cluster, OSM will configure the sidecar proxy on pods to allow ingress traffic to the service based on the ingress routing rules defined by the Kubernetes Ingress resource.

Note:
1. This behavior opens up HTTP-based access to any client that is not a part of the service mesh, not just ingress.
1. The ingress resource that allows external HTTP access to a particular service must be in the same namespace as that service.

### HTTP path matching semantics

The Kubernetes Ingress API allows specifying a [pathType](https://kubernetes.io/docs/concepts/services-networking/ingress/#path-types) for each HTTP `path` specified in an ingress rule. OSM enforces different HTTP path matching semantics depending on the `pathType` attribute specified. This allows OSM to operate with a number of different ingress controllers.

The following path matching semantics correspond to the value of the `pathType` attribute:

- `Exact`: With this path type, the `:path` header in the HTTP request is matched exactly to the `path` specified in the ingress rule.

- `Prefix`: With this path type, the `:path` header in the HTTP request is matched as an element wise prefix of the `path` specified in the ingress rule, as defined in the [Kubernetes ingress API specification](https://kubernetes.io/docs/concepts/services-networking/ingress/#path-types).

- `ImplementationSpecific`: With this path type, the `:path` header in the HTTP request is matched differently depending on the `path` specified in the ingress rule. If the specified `path` ~looks like a regex~ (has one of the following characters: `^$*+[]%|`), the `:path` header in the HTTP request is matched against the `path` specified in the ingress rule as a regex match using the [Google RE2 regex syntax](https://github.com/google/re2/wiki/Syntax). If the specified `path` ~does not look like a regex~, the `:path` header in the HTTP request is matched as a string prefix of the specified `path` in the ingress rule.

By default, if the `pathType` attribute is not set for a `path` in an ingress rule, OSM will default the `pathType` as `ImplementationSpecific`.

## Sample demo

### HTTP traffic with ingress

The following demo sends a request from an external IP to a httpbin service inside the cluster.

1. Install OSM.
    ```bash
    osm install
    ```

1. Install the [nginx ingress controller](https://kubernetes.github.io/ingress-nginx/deploy/#installation-guide)

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

1. Apply an ingress configuration yaml to expose the HTTP path `/status/200` on the `httpbin` service with `kubectl apply -f`:

    > Note: Use the appropriate ingress resource based on the desired API version.

    Ingress v1 resource:
    ```yaml
    apiVersion: networking.k8s.io/v1
    kind: Ingress
    metadata:
      name: httpbin-ingress
      namespace: httpbin
    spec:
      ingressClassName: nginx
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
        kubernetes.io/ingress.class: nginx
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

    Confirm that the httpbin-ingress has been successfully deployed.

    ```console
    $ kubectl get ingress -n httpbin
    NAME              CLASS    HOSTS   ADDRESS         PORTS   AGE
    httpbin-ingress   <none>   *       20.72.132.186   80      11h
    ```

1. Confirm that a request to the httpbin service from the external IP address of the Ingress resource succeeds (in this case, the external address would be `20.72.132.186`)

    ```bash
    curl http://<external-ip>/status/200
    ```

1. Update the existing ingress with a host specified by applying the following yaml, and confirm that the request succeeds:

    > Note: Use the appropriate ingress resource based on the desired API version.

    Ingress v1 resource:
    ```yaml
    apiVersion: networking.k8s.io/v1
    kind: Ingress
    metadata:
      name: httpbin-ingress
      namespace: httpbin
    spec:
      ingressClassName: nginx
      rules:
      - host: httpbin.com
        http:
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
        kubernetes.io/ingress.class: nginx
    spec:
      rules:
      - host: httpbin.com
        http:
          paths:
          - path: /status/200
            pathType: ImplementationSpecific # Must be one of: Exact, Prefix, ImplementationSpecific
            backend:
              serviceName: httpbin
              servicePort: 14001
    ```

    ```bash
    curl http://<external-ip>/status/200 -H "Host: httpbin.com"
    ```


## Sample Ingress Configurations

The following section describes sample ingress configurations used to expose services managed by OSM outside the cluster. The configuration might differ based on the ingress controller being used.

The example configurations describe how to expose HTTP and HTTPS routes for the `httpbin` service running on a pod with the service account `httpbin` on port `14001` in the `httpbin` namespace, outside the cluster. The ingress configuration will expose the HTTP path `/status/200` on the `httpbin` service.

Since OSM uses its own root certificate, the ingress controller must be provisioned with OSM's root certificate to be able to authenticate the certificate presented by backend servers when using HTTPS ingress. With `Tresor` as the certificate provider, OSM stores the CA root certificate in a Kubernetes secret named `osm-ca-bundle` with the key `ca.crt` in the namespace OSM is deployed (`osm-system` by default). When using other certificate providers such as `cert-manager.io` or `Hashicorp Vault`, the `osm-ca-bundle` secret must be created by the user with the base64 encoded root certificate stored as the value to the `ca.crt` attribute in the secret's data.

### Prerequisites
- Install [nginx ingress controller](https://kubernetes.github.io/ingress-nginx/deploy/#installation-guide)
- Specify the ingress controller as nginx using the annotation `kubernetes.io/ingress.class: nginx`.

For HTTPS ingress, additional annotations are required.
- Specify the backend protocol as HTTPS using the annotation `nginx.ingress.kubernetes.io/backend-protocol: "HTTPS"`.
- Specify the SAN to use to verify the HTTPS backend using the annotation `nginx.ingress.kubernetes.io/configuration-snippet`.
- Specify the secret corresponding to the root certificate using the annotation `nginx.ingress.kubernetes.io/proxy-ssl-secret`.
- Specify the passing of TLS Server Name Indication (SNI) to proxied HTTPS backends using the annotation `nginx.ingress.kubernetes.io/proxy-ssl-server-name`. This is optional.
- Enable SSL verification of backend service using the annotation `nginx.ingress.kubernetes.io/proxy-ssl-verify`.

### Examples

1. HTTP ingress resource with wildcard host:
    ```yaml
    apiVersion: networking.k8s.io/v1beta1
    kind: Ingress
    metadata:
      name: httpbin-ingress
      namespace: httpbin
      annotations:
        kubernetes.io/ingress.class: nginx
    spec:
      rules:
      - http:
          paths:
          - path: /status/200
            backend:
              serviceName: httpbin
              servicePort: 14001
    ```

    Accessing the service:
    ```bash
    curl http://<external-ingress-ip>/status/200
    ```

1. HTTP ingress resource with host specified:
    ```yaml
    apiVersion: networking.k8s.io/v1beta1
    kind: Ingress
    metadata:
      name: httpbin-ingress
      namespace: httpbin
      annotations:
        kubernetes.io/ingress.class: nginx
    spec:
      rules:
      - host: httpbin.com
        http:
          paths:
          - path: /status/200
            backend:
              serviceName: httpbin
              servicePort: 14001
    ```

    Accessing the service:
    ```bash
    curl http://<external-ingress-ip>/status/200 -H "Host: httpbin.com"
    ```

1. HTTPS ingress with host specified:

    Here, the requests to the backend are proxied over HTTPS. As a result, the root CA certificate used to verify the certificate presented by the backend must be configured, along with other SSL parameters.
    ```yaml
    apiVersion: networking.k8s.io/v1beta1
    kind: Ingress
    metadata:
      name: httpbin-ingress
      namespace: httpbin
      annotations:
        kubernetes.io/ingress.class: nginx
        nginx.ingress.kubernetes.io/backend-protocol: "HTTPS"
        # proxy_ssl_name for a service is of the form <service-account>.<namespace>.cluster.local
        nginx.ingress.kubernetes.io/configuration-snippet: |
          proxy_ssl_name "httpbin.httpbin.cluster.local";
        # k8s secret with CA certificate stored with key ca.crt
        nginx.ingress.kubernetes.io/proxy-ssl-secret: "osm-system/osm-ca-bundle"
        nginx.ingress.kubernetes.io/proxy-ssl-server-name: "on" # optional
        nginx.ingress.kubernetes.io/proxy-ssl-verify: "on"
    spec:
      rules:
      - host: httpbin.com
        http:
          paths:
          - path: /status/200
            backend:
              serviceName: httpbin
              servicePort: 14001
    ```
    Accessing the service:
    ```bash
    curl http://<external-ingress-ip>/status/200 -H "Host: httpbin.com"
    ```

### Using Gloo API Gateway (experimental)

[Gloo API Gateway][5] is an Envoy-powered API gateway that can run in `Ingress` mode or full-blow `Gateway` mode. In this document, we show the `Ingress` approach, but you can refer to the [Gloo documentation][5] for more in depth functionality enabled by Gloo.

Install Gloo in `Ingress` mode:

```bash
glooctl install ingress
```

For Gloo's ingress, we don't need any additional annotations, however, the `kubernetes.io/ingress.class: gloo` annotation is recommended. With Gloo, we configure the `Upstream` objects with the appropriate trust authority. In Gloo, the `Upstream` object represents a routable target (Kubernetes Service, Consul Service, Cloud Function, etc).

To prepare the root certificate, we must do something similar to what we do for the Azure Application Gateway.

```bash
kubectl get secret/osm-ca-bundle -n osm-system -o jsonpath="{.data['ca\.crt']}" | base64 -d > osm-c-bundlea.pem

glooctl create secret tls --name osm-ca-bundle --rootca osm-c-bundlea.pem
```

Next we could use an Ingress file like this:

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
  - host: httpbin.httpbin.svc.cluster.local
    http:
      paths:
      - path: /status/200
        backend:
          serviceName: httpbin
          servicePort: 14001
```

Lastly, we configure the `Upstream` object to use OSM's root ca bundle:

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

At this point you can call your Ingress endpoint and get HTTPS traffic from the edge to your OSM service. As a convenience, you can run the following to get your ingress hostname/IP:

```bash
glooctl proxy url --name ingress-proxy
```

[1]: https://github.com/openservicemesh/osm/blob/release-v0.8/README.md
[2]: https://kubernetes.github.io/ingress-nginx/
[3]: https://azure.github.io/application-gateway-kubernetes-ingress/
[4]: https://github.com/Azure/application-gateway-kubernetes-ingress/blob/master/docs/annotations.md#appgw-trusted-root-certificate
[5]: https://docs.solo.io/gloo/latest/
