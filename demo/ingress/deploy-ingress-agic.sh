#!/bin/bash



# This script deploys a Kubernetes Ingress resource.
# This is a helper script used to ease the demonstration of OSM.



set -aueo pipefail

# shellcheck disable=SC1091
source .env


# Empty string is the default value, which would disable Ingress integration testing.
INGRESS_HOSTNAME="${INGRESS_HOSTNAME:-bookstore.osm.contoso.com}"


echo "Create Bookstore Ingress Resource"
kubectl apply -f - <<EOF
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: bookstore-v1
  namespace: "${BOOKSTORE_NAMESPACE}"
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
spec:
  rules:
  - host: "v1.${INGRESS_HOSTNAME}"
    http:
      paths:
      - path: /
        backend:
          serviceName: bookstore-v1
          servicePort: 80

  - host: "v2.${INGRESS_HOSTNAME}"
    http:
      paths:
      - path: /
        backend:
          serviceName: bookstore-v2
          servicePort: 80

  - http:
      paths:
      - path: /
        backend:
          serviceName: bookstore-v1
          servicePort: 80
EOF
