#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

echo -e "Deploy $PROMETHEUS_SVC monitoring service"
cat <<EOF | kubectl apply -f -
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: "$PROMETHEUS_SVC-deployment"
  namespace: "$PROMETHEUS_NAMESPACE"
spec:
  replicas: 1
  strategy:
    type: Recreate
  template:
    metadata:
      labels:
        app: "$PROMETHEUS_SVC-server"
    spec:
      serviceAccountName: "$PROMETHEUS_SVC-serviceaccount"
      containers:
        - name: prometheus
          image: prom/prometheus:v2.2.1
          args:
            - "--config.file=/etc/$PROMETHEUS_SVC/prometheus.yml"
            - "--storage.tsdb.path=/$PROMETHEUS_SVC/"
          ports:
            - containerPort: 9090
          volumeMounts:
            - name: "$PROMETHEUS_SVC-config-volume"
              mountPath: /etc/$PROMETHEUS_SVC/
            - name: "$PROMETHEUS_SVC-storage-volume"
              mountPath: /$PROMETHEUS_SVC/
      volumes:
        - name: "$PROMETHEUS_SVC-config-volume"
          configMap:
            defaultMode: 420
            name: "$PROMETHEUS_SVC-server-conf"

        - name: "$PROMETHEUS_SVC-storage-volume"
          emptyDir: {}
---

apiVersion: v1
kind: Service
metadata:
  name: "$PROMETHEUS_SVC-service"
  namespace: "$PROMETHEUS_NAMESPACE"
  annotations:
      prometheus.io/scrape: 'true'
      prometheus.io/port:   '9090'
spec:
  selector:
    app: "$PROMETHEUS_SVC-server"
  type: NodePort
  ports:
    - port: 8080
      targetPort: 9090
EOF
      