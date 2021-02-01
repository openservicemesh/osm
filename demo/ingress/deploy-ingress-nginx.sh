#!/bin/bash



# This script deploys a Kubernetes Ingress resource for Nginx.
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
  namespace: $BOOKSTORE_NAMESPACE
  annotations:
    kubernetes.io/ingress.class: nginx
spec:
  rules:
  - http:
      paths:
      - path: /books-bought
        backend:
          serviceName: bookstore-v1
          servicePort: 80
EOF