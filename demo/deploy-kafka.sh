#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env
DEPLOY_ON_OPENSHIFT="${DEPLOY_ON_OPENSHIFT:-false}"
USE_PRIVATE_REGISTRY="${USE_PRIVATE_REGISTRY:-false}"
MESH_NAME="${MESH_NAME:-osm}"

kubectl create ns kafka 

bin/osm namespace add --mesh-name "$MESH_NAME" kafka

bin/osm metrics enable --namespace kafka

helm install kafka bitnami/kafka --set replicaCount=3 --set zookeeper.enabled=false --set zookeeperChrootPath='/kafka-root' --set serviceAccount.create=true --set serviceAccount.name=kafka --namespace kafka --set "externalZookeeper.servers={kafka-zookeeper-0.kafka-zookeeper-headless.zookeeper.svc.cluster.local,kafka-zookeeper-1.kafka-zookeeper-headless.zookeeper.svc.cluster.local,kafka-zookeeper-2.kafka-zookeeper-headless.zookeeper.svc.cluster.local}"

if [ "$DEPLOY_ON_OPENSHIFT" = true ] ; then
    oc adm policy add-scc-to-user privileged -z "kafka" -n "kafka"
    if [ "$USE_PRIVATE_REGISTRY" = true ]; then
        oc secrets link "kafka" "$CTR_REGISTRY_CREDS_NAME" --for=pull -n "kafka"
    fi
fi

kubectl apply -nkafka -f - <<EOF
apiVersion: specs.smi-spec.io/v1alpha4
kind: TCPRoute
metadata:
  name: kafka
  namespace: kafka
spec:
  matches:
    ports:
    - 9092
---
apiVersion: specs.smi-spec.io/v1alpha4
kind: TCPRoute
metadata:
  name: kafka-internal
  namespace: kafka
spec:
  matches:
    ports:
    - 9092
    - 9093
---
kind: TrafficTarget
apiVersion: access.smi-spec.io/v1alpha3
metadata:
  name: kafka
  namespace: kafka
spec:
  destination:
    kind: ServiceAccount
    name: kafka
    namespace: kafka
  rules:
  - kind: TCPRoute
    name: kafka
  sources:
  - kind: ServiceAccount
    name: default
    namespace: kafka
---
kind: TrafficTarget
apiVersion: access.smi-spec.io/v1alpha3
metadata:
  name: kafka-internal
  namespace: kafka
spec:
  destination:
    kind: ServiceAccount
    name: kafka
    namespace: kafka
  rules:
  - kind: TCPRoute
    name: kafka-internal
  sources:
  - kind: ServiceAccount
    name: kafka
    namespace: kafka
EOF

# Use these commands to test out Kafka
# kubectl run kafka-client --restart='Never' --image docker.io/bitnami/kafka:3.1.0-debian-10-r60 --namespace kafka --command -- sleep infinity
# kubectl exec --tty -i kafka-client --namespace kafka -- bash
# kafka-console-producer.sh --broker-list kafka-0.kafka-headless.kafka.svc.cluster.local:9092 --topic test
# kafka-console-consumer.sh --bootstrap-server kafka.kafka.svc.cluster.local:9092 --topic test --from-beginning
