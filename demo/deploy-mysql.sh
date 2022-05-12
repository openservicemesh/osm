#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env
DEPLOY_ON_OPENSHIFT="${DEPLOY_ON_OPENSHIFT:-false}"
USE_PRIVATE_REGISTRY="${USE_PRIVATE_REGISTRY:-false}"

kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: mysql
  namespace: $BOOKWAREHOUSE_NAMESPACE
---
apiVersion: v1
kind: Service
metadata:
  name: mysql
  namespace: $BOOKWAREHOUSE_NAMESPACE
spec:
  ports:
  - port: 3306
    targetPort: 3306
    name: client
    appProtocol: tcp-server-first
  selector:
    app: mysql
  clusterIP: None
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
  namespace: $BOOKWAREHOUSE_NAMESPACE
spec:
  serviceName: mysql
  replicas: 1
  selector:
    matchLabels:
      app: mysql
  template:
    metadata:
      labels:
        app: mysql
    spec:
      serviceAccountName: mysql
      nodeSelector:
        kubernetes.io/os: linux
      containers:
      - image: mysql:5.6
        name: mysql
        env:
        - name: MYSQL_ROOT_PASSWORD
          value: mypassword
        - name: MYSQL_DATABASE
          value: booksdemo
        ports:
        - containerPort: 3306
          name: mysql
        volumeMounts:
        - mountPath: /mysql-data
          name: data
        readinessProbe:
          tcpSocket:
            port: 3306
          initialDelaySeconds: 15
          periodSeconds: 10
      volumes:
        - name: data
          emptyDir: {}
      imagePullSecrets:
        - name: $CTR_REGISTRY_CREDS_NAME
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 250M
---
kind: TrafficTarget
apiVersion: access.smi-spec.io/v1alpha3
metadata:
  name: mysql
  namespace: $BOOKWAREHOUSE_NAMESPACE
spec:
  destination:
    kind: ServiceAccount
    name: mysql
    namespace: $BOOKWAREHOUSE_NAMESPACE
  rules:
  - kind: TCPRoute
    name: mysql
  sources:
  - kind: ServiceAccount
    name: bookwarehouse
    namespace: $BOOKWAREHOUSE_NAMESPACE
---
apiVersion: specs.smi-spec.io/v1alpha4
kind: TCPRoute
metadata:
  name: mysql
  namespace: $BOOKWAREHOUSE_NAMESPACE
spec:
  matches:
    ports:
    - 3306
EOF

if [ "$DEPLOY_ON_OPENSHIFT" = true ] ; then
    oc adm policy add-scc-to-user privileged -z "mysql" -n "$BOOKWAREHOUSE_NAMESPACE"
    if [ "$USE_PRIVATE_REGISTRY" = true ]; then
        oc secrets link "mysql" "$CTR_REGISTRY_CREDS_NAME" --for=pull -n "$BOOKWAREHOUSE_NAMESPACE"
    fi
fi

