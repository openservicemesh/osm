#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

echo "Create Bookstore Ingress Resource"
kubectl apply -f - <<EOF
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: bookstore-v1
  namespace: ${BOOKSTORE_NAMESPACE}

  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    appgw.ingress.kubernetes.io/backend-protocol: "https"
    appgw.ingress.kubernetes.io/appgw-trusted-root-certificate: "osm-root-cert"
    appgw.ingress.kubernetes.io/backend-hostname: "bookstore-v1.bookstore-ns.svc.cluster.local"
    openservicemesh.io/monitor: enabled

spec:

  rules:

  - http:
      paths:

      - path: /books-bought
        backend:
          serviceName: bookstore-v1
          servicePort: 80

      - path: /buy-a-book/new
        backend:
          serviceName: bookstore-v1
          servicePort: 80
EOF

echo "Create Bookstore Ingress Resource"
kubectl apply -f - <<EOF
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: bookstore-v1
  namespace: ${BOOKSTORE_NAMESPACE}

  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    appgw.ingress.kubernetes.io/backend-protocol: "https"
    appgw.ingress.kubernetes.io/appgw-trusted-root-certificate: "osm-ca-bundle"
    appgw.ingress.kubernetes.io/backend-hostname: "bookstore-v1.bookstore-ns.svc.cluster.local"
    openservicemesh.io/monitor: enabled

spec:

  rules:

  - host: bookstore.contoso.com
    http:
      paths:
      - path: /*
        backend:
          serviceName: bookstore-v1
          servicePort: 80
EOF


echo "Create Bookbuyer Ingress Resource"
kubectl apply -f - <<EOF
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: bookbuyer
  namespace: ${BOOKBUYER_NAMESPACE}
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    appgw.ingress.kubernetes.io/backend-protocol: "https"
    appgw.ingress.kubernetes.io/appgw-trusted-root-certificate: "osm-ca-bundle"
    appgw.ingress.kubernetes.io/backend-hostname: "bookbuyer-v1.bookbuyer-ns.svc.cluster.local"
    openservicemesh.io/monitor: enabled
spec:
  rules:
  - host: bookbuyer.contoso.com
    http:
      paths:
      - path: /*
        backend:
          serviceName: bookbuyer
          servicePort: 80
EOF
