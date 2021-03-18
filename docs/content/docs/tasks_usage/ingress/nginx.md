---
title: "Using Nginx Ingress Controller with OSM"
description: "This document provides step by step instructions on installing and integrationg Nginx Ingress Controller with Open Service Mesh."
type: docs
aliases: ["Nginx"]
weight: 2
release: 0.8.0
---

Nginx is a popular layer 7 reverse proxy and load balancer. Nginx Ingress Controller is the Kubernetes component, which continuously configures Nginx to expose Kubernetes Services to the Internet and other external to the cluster networks.

Excellent documentation is already [avalible on the Nginx website](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-helm/#adding-the-helm-repository). The following 2 shell commands will use Helm to install Nginx:

```bash
helm repo add nginx-stable https://helm.nginx.com/stable
helm repo update
helm install nginx-stable/nginx-ingress --generate-name
```

View the newly installed ingress controller pod:
```bash
kubectl get pods | grep nginx
```

The example below uses OSM's `bookstore` app to illustrate exposing a Kubernetes service to the Internet:

1. Create bookstore service:
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

2. Create bookstore pod:
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
  serviceAccountName: bookstore
  containers:
  - name: bookstore
    image: openservicemesh/bookstore:v0.8.0
    ports:
      - containerPort: 14001
    command: ["/bookstore"]
    args: ["--port", "14001"]
EOF
```

3. Expose the bookstore service to the Internet:
```bash
kubectl apply -f - <<EOF
---
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: bookstore-ingress
  annotations:
    kubernetes.io/ingress.class: nginx

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

4. View Nginx logs:
```bash
POD=$(kubectl get pods | grep 'nginx-ingress' | awk '{print $1}')

kubectl logs $POD -f
```

5. View Nginx Services:
```bash
kubectl get services
```

```
NAME                                     TYPE           CLUSTER-IP     EXTERNAL-IP    PORT(S)                      AGE
nginx-ingress-1616041155-nginx-ingress   LoadBalancer   10.0.120.194   1.2.3.4        80:32237/TCP,443:31563/TCP   23m
```

6. Test your service via the external IP address:

If you have DNS setup already:
```bash
curl http://bookstore.contoso.com/
```

Alternatively:
```bash
curl -H 'Host: bookstore.contoso.com' http://1.2.3.4/
```
