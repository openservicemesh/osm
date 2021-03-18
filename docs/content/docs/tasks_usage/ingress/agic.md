---
title: "Using Azure Application Gateway Ingress Controller with OSM"
description: "This document provides step by step instructions on installing and integrationg Azure Application Gateway Ingress Controller with Open Service Mesh."
type: docs
aliases: ["AGIC"]
weight: 2
release: 0.8.0
---

The [Azure Application Gateway Ingress Controller (AGIC)](https://docs.microsoft.com/en-us/azure/application-gateway/ingress-controller-overview)
is a Kubernetes controller for the Azure Application Gateway.
AGIC works with in tandem with
[Azure Application Gateway](https://docs.microsoft.com/en-us/azure/application-gateway/overview)
to provide ingress for Kubernetes clusters on Azure.
AGIC is installed on the cluster. The App Gateway is outside of the Kubernetes cluster
and exposed to the Internet via a public IP address.
AGIC continuously monitors the Kubernetes cluster for changes.
Any updates of services and pods are reflected in Azure Application Gateway's configuration.
App Gateway's backend pools are updated with the latest available Endpoints (Pod IP)
addresses from the Kubernetes cluster.

Excellent and thorough documentation is already [avalible on AGIC's website](https://docs.microsoft.com/en-us/azure/application-gateway/ingress-controller-overview).

The example below uses OSM's `bookstore` app to illustrate exposing a Kubernetes service to the Internet:

2. Create bookstore service:
```bash
kubectl apply -f - <<EOF
---
apiVersion: v1
kind: Service
metadata:
  name: bookstore
  labels:
    app: bookstore
spec:
  selector:
    app: bookstore
  ports:
  - port: 14001
EOF
```

3. Create bookstore pod:
```bash
kubectl apply -f - <<EOF
---
apiVersion: v1
kind: Pod
metadata:
  name: bookstore
  labels:
    app: bookstore
spec:
  containers:
  - name: bookstore
    image: openservicemesh/bookstore:v0.8.0
    ports:
      - containerPort: 14001
    command: ["/bookstore"]
    args: ["--port", "14001"]
EOF
```

4. Expose the bookstore service to the Internet:
```bash
kubectl apply -f - <<EOF
---
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: bookstore-ingress
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway

spec:

  rules:
    - host: bookstore.contoso.com
      http:
        paths:
        - path: /
          backend:
            serviceName: bookstore
            servicePort: 14001

  backend:
    serviceName: bookstore
    servicePort: 14001
EOF
```

5. Test your service via the public IP address of App Gateway:

If you have DNS setup already:
```bash
curl http://bookstore.contoso.com/
```

Alternatively - get the public IP address of Azure Application Gateway (1.2.3.4 for instance):
```bash
curl -H 'Host: bookstore.contoso.com' http://1.2.3.4/
```

# Troubleshooting
  - [AGIC Troubleshooting Documentation](https://docs.microsoft.com/en-us/azure/application-gateway/ingress-controller-troubleshoot)
  - [Additional troubleshooting tools are available on AGIC's GitHub repo](https://github.com/Azure/application-gateway-kubernetes-ingress/blob/master/docs/troubleshootings/troubleshooting-installing-a-simple-application.md)
