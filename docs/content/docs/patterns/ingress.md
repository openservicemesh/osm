---
title: "Ingress"
description: "How to expose HTTP and HTTPS routes outside the cluster to services within the cluster using Kubernetes Ingress"
type: docs
---

# Exposing services outside the cluster using Ingress
This document describes how to expose HTTP and HTTPS routes outside the cluster to services within the cluster using Kubernetes Ingress.

## Prerequisites
- An instance of OSM must be running in the cluster.
- The service needing to be exposed using Ingress needs to belong to a namespace monitored by OSM. Refer to the [Readme][1] for details.
- The ingress resource must belong to the same namespace as the backend service.
- A sidecar must be injected to the pod hosting the service, either using automatic sidecar injection or by manually annotating the pod spec for sidecar injection. Refer to the [Readme][1] for details.
- An ingress controller must be running in the cluster.

## Exposing an HTTP or HTTPS service using Ingress
A service can expose HTTP or HTTPS routes outside the cluster using Kubernetes Ingress along with an ingress controller. Once an ingress resource is configured to expose HTTP routes outside the cluster to a service within the cluster, OSM will configure the sidecar proxy on pods to allow ingress traffic to the service based on the ingress routing rules defined by the Kubernetes Ingress resource. Keep in mind, this behavior opens up HTTP-based access to any client that is not a part of the service mesh, not just ingress.

HTTPS ingress support is experimental. OSM supports one way TLS authentication to backend services.

By default, OSM configures HTTP as the backend protocol for services when an ingress resource is applied with a backend service that belongs to the mesh. A mesh-wide configuration setting in OSM's `osm-config` ConfigMap enables configuring ingress with the backend protocol to be HTTPS. HTTPS ingress can be enabled by updating the `osm-config` ConfigMap in `osm-controller`'s namespace (`osm-system` by default).

Patch the ConfigMap by setting `use_https_ingress: "true"`.
```bash
kubectl patch ConfigMap osm-config -n osm-system -p '{"data":{"use_https_ingress":"true"}}' --type=merge
```

## Ingress controller compatibility
Ingress in OSM is compatible with the following ingress controllers.
- [Nginx Ingress Controller][2]
- [Azure Application Gateway Ingress Controller][3]
- [Gloo API Gateway][5]

Other ingress controllers might also work as long as they use Kubernetes Ingress resource and allow provisioning a custom root certificate for HTTPS backend server certificate validation.

## Ingress configurations
The following section describes sample ingress configurations used to expose services managed by OSM outside the cluster. The configuration might differ based on the ingress controller being used.

The example configurations describe how to expose HTTP and HTTPS routes for the `bookstore-v1` service running on port `80` in the `bookstore` namespace, outside the cluster. The ingress configuration will expose the HTTP path `/books-bought` on the `bookstore-v1` service.

Since OSM uses its own root certificate, the ingress controller must be provisioned with OSM's root certificate to be able to authenticate the certificate presented by backend servers when using HTTPS ingress. OSM stores the CA root certificate in a Kubernetes secret named `osm-ca-bundle` with the key `ca.crt` in the namespace OSM is deployed (`osm-system` by default).

### Using Nginx Ingress Controller
An ingress configuration yaml with [Nginx Ingress Controller][2] for the `bookstore-v1` service described above would look as follows.

- Specify the ingress controller as nginx using the annotation `kubernetes.io/ingress.class: nginx`.

For HTTPS ingress, additional annotations are required.
- Specify the backend protocol as HTTPS using the annotation `nginx.ingress.kubernetes.io/backend-protocol: "HTTPS"`.
- Specify the hostname of the service using the annotation `nginx.ingress.kubernetes.io/configuration-snippet`.
- Specify the secret corresponding to the root certificate using the annotation `nginx.ingress.kubernetes.io/proxy-ssl-secret`.
- Specify the passing of TLS Server Name Indication (SNI) to proxied HTTPS backends using the annotation `nginx.ingress.kubernetes.io/proxy-ssl-server-name`. This is optional.
- Enable SSL verification of backend service using the annotation `nginx.ingress.kubernetes.io/proxy-ssl-verify`.

The host defined by `spec.rules.host` field is optional.

HTTP ingress sample configuration:
```yaml
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  name: bookstore-v1
  namespace: bookstore
  annotations:
    kubernetes.io/ingress.class: nginx
spec:
  rules:
  - host: bookstore-v1.bookstore.svc.cluster.local
    http:
      paths:
      - path: /books-bought
        backend:
          serviceName: bookstore-v1
          servicePort: 80
```

HTTPS ingress sample configuration:
```yaml
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  name: bookstore-v1
  namespace: bookstore
  annotations:
    kubernetes.io/ingress.class: nginx
    nginx.ingress.kubernetes.io/backend-protocol: "HTTPS"
    nginx.ingress.kubernetes.io/configuration-snippet: |
      proxy_ssl_name "bookstore-v1.bookstore.svc.cluster.local";
    nginx.ingress.kubernetes.io/proxy-ssl-secret: "osm-system/osm-ca-bundle"
    nginx.ingress.kubernetes.io/proxy-ssl-server-name: "on" # optional
    nginx.ingress.kubernetes.io/proxy-ssl-verify: "on"
spec:
  rules:
  - host: bookstore-v1.bookstore.svc.cluster.local
    http:
      paths:
      - path: /books-bought
        backend:
          serviceName: bookstore-v1
          servicePort: 80
```

### Using Azure Application Gateway Ingress Controller
An ingress configuration yaml with [Azure Application Gateway Ingress Controller][3] for the `bookstore-v1` service described above would look as follows.

- Specify the ingress controller as Azure Application Gateway using the annotation `kubernetes.io/ingress.class: azure/application-gateway`.

For HTTPS ingress, additional annotations are required.
- Specify the backend protocol as HTTPS using the annotation `appgw.ingress.kubernetes.io/backend-protocol: "https"`.
- Specify the root certificate name added to Azure Application Gateway corresponding to OSM's root certificate using the annotation `appgw.ingress.kubernetes.io/appgw-trusted-root-certificate`. Refer to the document on [adding trusted root certificates to Azure Application Gateway][4].
    ```bash
    # Download "osm-ca-bundle" certificate bundle from the cluster
    kubectl get secret -n osm-system osm-ca-bundle -o json | jq -r '.data["ca.crt"]' | base64 -d > osm-ca-bundle.pem

    # Upload osm-ca-bundle to the Application Gateway
    az network application-gateway root-cert create \
    --gateway-name <gateway-name> \
    -g <resource-group> \
    -n osm-ca-bundle \
    --cert-file osm-ca-bundle.pem
    ```
- Specify the hostname for the backend service using the annotation `appgw.ingress.kubernetes.io/backend-hostname`.

The host defined by `spec.rules.host` field is optional and skipped in the example below.

HTTP ingress sample configuration:
```yaml
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  name: bookstore-v1
  namespace: bookstore
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
spec:
  rules:
  - http:
      paths:
      - path: /books-bought
        backend:
          serviceName: bookstore-v1
          servicePort: 80
```

HTTPS ingress sample configuration:
```yaml
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  name: bookstore-v1
  namespace: bookstore
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    appgw.ingress.kubernetes.io/backend-protocol: "https"
    appgw.ingress.kubernetes.io/appgw-trusted-root-certificate: "osm-ca-bundle"
    appgw.ingress.kubernetes.io/backend-hostname: "bookstore-v1.bookstore.svc.cluster.local"
spec:
  rules:
  - http:
      paths:
      - path: /books-bought
        backend:
          serviceName: bookstore-v1
          servicePort: 8080 # Note: port 80 cannot be used for HTTPS ingress with Azure Application Gateway ingress
```
### Using Gloo API Gateway

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
  name: bookstore-v1
  namespace: bookstore
  annotations:
    kubernetes.io/ingress.class: gloo
spec:
  rules:
  - host: bookstore-v1.bookstore.svc.cluster.local
    http:
      paths:
      - path: /books-bought
        backend:
          serviceName: bookstore-v1
          servicePort: 80
```

Lastly, we configure the `Upstream` object to use OSM's root ca bundle:

```yaml
apiVersion: gloo.solo.io/v1
kind: Upstream
metadata:
  name: bookstore-bookstore-80
  namespace: gloo-system
spec:
  sslConfig:
    sni: "bookstore-v1.bookstore.svc.cluster.local"
    secretRef:
      name: osm-ca-bundle
      namespace: gloo-system
  kube:
    selector:
      app: bookstore
    serviceName: bookstore
    serviceNamespace: bookstore
    servicePort: 80

```

At this point you can call your Ingress endpoint and get HTTPS traffic from the edge to your OSM service. As a convenience, you can run the following to get your ingress hostname/IP:

```bash
glooctl proxy url --name ingress-proxy
```

[1]: https://github.com/openservicemesh/osm/blob/main/README.md
[2]: https://kubernetes.github.io/ingress-nginx/
[3]: https://azure.github.io/application-gateway-kubernetes-ingress/
[4]: https://github.com/Azure/application-gateway-kubernetes-ingress/blob/master/docs/annotations.md#appgw-trusted-root-certificate
[5]: https://docs.solo.io/gloo/latest/
