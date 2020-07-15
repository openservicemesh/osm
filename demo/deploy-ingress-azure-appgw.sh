#!/bin/bash



# This script deploys a Kubernetes Ingress resource.
# This is a helper script used to ease the demonstration of OSM.



set -aueo pipefail

# shellcheck disable=SC1091
source .env


echo "Create Bookstore Ingress Resource"
kubectl apply -f - <<EOF
apiVersion: extensions/v1beta1
kind: Ingress

metadata:
  name: bookstore-v1
  namespace: "${BOOKSTORE_NAMESPACE}"

  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    appgw.ingress.kubernetes.io/backend-protocol: "https"
    appgw.ingress.kubernetes.io/appgw-trusted-root-certificate: "osm-ca-bundle"
    appgw.ingress.kubernetes.io/backend-hostname: "bookstore-v1.bookstore-ns.svc.cluster.local"

spec:

  rules:

  - host: bookstore.contoso.com
    http:
      paths:

      - path: /*
        backend:
          serviceName: bookstore-v1
          servicePort: 80

      - path: /books-bought
        backend:
          serviceName: bookstore-v1
          servicePort: 80

      - path: /buy-a-book/new
        backend:
          serviceName: bookstore-v1
          servicePort: 80

EOF
