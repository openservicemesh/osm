#!/bin/bash

POD_CIDR=$(kubectl cluster-info dump | grep -m 1 -Eo -- '--cluster-cidr=[0-9./]+' | cut -d= -f2)

# Apply a spec with invalid IP to figure out the valid IP range
SERVICE_CIDR=$(kubectl apply -f - <<EOF 2>&1 | sed 's/.*valid IPs is //'
apiVersion: v1
kind: Service
metadata:
  name: fake
  namespace: default
spec:
  clusterIP: 1.1.1.1
  ports:
    - port: 443
EOF
)
echo "$POD_CIDR,$SERVICE_CIDR"
