# Exposing services outside the cluster using Ingress
This document describes the workflow to expose HTTPS routes outside the cluster to services within the cluster using Kubernetes Ingress.

OSM supports one way TLS authentication to backend services.

## Prerequisites
- An instance of OSM must be running in the cluster.
- The service needing to be exposed using Ingress needs to belong to a namespace monitored by OSM. Refer to the [Readme][1] for details.
- The ingress resource must belong to the same namespace as the backend service.
- A sidecar must be injected to the pod hosting the service, either using automatic sidecar injection or by manually annotating the pod spec for sidecar injection. Refer to the [Readme][1] for details.
- An ingress controller must be running in the cluster.
- The service must be an HTTPS service whose certificates are provisioned by OSM. OSM uses its own root certificate and a custom Certificate Authority (CA) for issuing certificates to services.

## Exposing an HTTPS service using Ingress
A service can expose HTTPS routes outside the cluster using Kubernetes Ingress along with an ingress controller. Once an ingress resource is configured to expose HTTPS routes outside the cluster to a service within the cluster, OSM will configure the sidecar proxy on pods to allow ingress traffic to the service based on the ingress routing rules defined by the Kubernetes Ingress resource.

## Ingress controller compatibility
Ingress in OSM is compatible with the following ingress controllers.
- [Nginx Ingress Controller][2]
- [Azure Application Gateway Ingress Controller][3]

Other ingress controllers might also work as long as they allow provisioning a custom root certificate for backend server certificate validation.

## Ingress configurations
The following section describes sample ingress configurations used to expose services managed by OSM outside the cluster.  Different ingress controllers require different configurations.

The example configurations describe how to expose HTTPS routes for the `bookstore-v1` HTTPS service running on port `8080` in the `bookstore-ns` namespace, outside the cluster. The ingress configuration will expose the HTTPS path `/books-bought` on the `bookstore-v1` service.

Since OSM uses its own root certificate, the ingress controller must be provisioned with OSM's root certificate to be able to authenticate the certificate presented by backend servers. OSM stores the CA root certificate in a Kubernetes secret called `osm-ca-bundle` with the key `ca.crt` in the namespace OSM is deployed (`osm-system` by default).

### Using Nginx Ingress Controller
An ingress configuration yaml with [Nginx Ingress Controller][2] for the `bookstore-v1` service described above would look as follows.

- Specify the ingress controller as nginx using the annotation `kubernetes.io/ingress.class: nginx`.
- Specify the backend protocol as HTTPS using the annotation `nginx.ingress.kubernetes.io/backend-protocol: "HTTPS"`.
- Specify the hostname of the service using the annotation `nginx.ingress.kubernetes.io/configuration-snippet`.
- Specify the secret corresponding to the root certificate using the annotation `nginx.ingress.kubernetes.io/proxy-ssl-secret`.
- Enable SSL verification of backend service using the annotation `nginx.ingress.kubernetes.io/proxy-ssl-verify`.

The host defined by `spec.rules.host` field is optional.
```
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: bookstore-v1
  namespace: bookstore-ns
  annotations:
    kubernetes.io/ingress.class: nginx
    nginx.ingress.kubernetes.io/backend-protocol: "HTTPS"
    nginx.ingress.kubernetes.io/configuration-snippet: |
      proxy_ssl_name "bookstore-v1.bookstore-ns.svc.cluster.local";
    nginx.ingress.kubernetes.io/proxy-ssl-secret: "osm-system/osm-ca-bundle"
    nginx.ingress.kubernetes.io/proxy-ssl-verify: "on"
spec:
  rules:
  - host: bookstore-v1.bookstore-ns.svc.cluster.local
    http:
      paths:
      - path: /books-bought
        backend:
          serviceName: bookstore-v1
          servicePort: 8080
```

### Using Azure Application Gateway Ingress Controller
An ingress configuration yaml with [Azure Application Gateway Ingress Controller][3] for the `bookstore-v1` service described above would look as follows.

- Specify the ingress controller as Azure Application Gateway using the annotation `kubernetes.io/ingress.class: azure/application-gateway`.
- Specify the backend protocol as HTTPS using the annotation `appgw.ingress.kubernetes.io/backend-protocol: "https"`.
- Specify the root certificate name added to Azure Application Gateway corresponding to OSM's root certificate using the annotation `appgw.ingress.kubernetes.io/appgw-trusted-root-certificate`. Refer to the document on [adding trusted root certificates to Azure Application Gateway][4].
- Specify the hostname for the backend service using the annotation `appgw.ingress.kubernetes.io/backend-hostname`.

The host defined by `spec.rules.host` field is optional and skipped in the example below.

```
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: bookstore-v1
  namespace: bookstore-ns
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    appgw.ingress.kubernetes.io/backend-protocol: "https"
    appgw.ingress.kubernetes.io/appgw-trusted-root-certificate: "osm-ca-bundle"
    appgw.ingress.kubernetes.io/backend-hostname: "bookstore-v1.bookstore-ns.svc.cluster.local"
spec:
  rules:
  - http:
      paths:
      - path: /books-bought
        backend:
          serviceName: bookstore-v1
          servicePort: 8080
```

[1]: https://github.com/open-service-mesh/osm/blob/main/README.md
[2]: https://kubernetes.github.io/ingress-nginx/
[3]: https://azure.github.io/application-gateway-kubernetes-ingress/
[4]: https://github.com/Azure/application-gateway-kubernetes-ingress/blob/master/docs/annotations.md#appgw-trusted-root-certificate
