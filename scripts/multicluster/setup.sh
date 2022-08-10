#!/bin/bash

# This script sets up a demo multicluster environment for OSM development.
# It deploys the sample bookstore application on cluster c1 and c2. C2 exports its
# bookstore service via ServiceExport and c1 imports it via ServiceImport. A new EndpointSlice
# resource on c1 is created to store accessible endpoint from c2.

# shellcheck disable=SC2128
# shellcheck disable=SC2086
cd "$(dirname ${BASH_SOURCE})"
set -e

c1=${CLUSTER1:-c1}
c2=${CLUSTER2:-c2}
k1="kubectl --context ${c1}"
k2="kubectl --context ${c2}"
app_port=14001

# ==== 1. Setup CRDs

${k1} apply -f ./crd -f ./rbac
${k2} apply -f ./crd -f ./rbac

# ==== 2. Deploy applications
for ctx in ${c1} ${c2}; do
    kubectl config use-context "${ctx}"

    echo "[${ctx}] Installing osm"
    osm install

    echo "[${ctx}] Deploying demo application"
    kubectl create namespace bookstore
    kubectl create namespace bookbuyer
    kubectl create namespace bookthief
    kubectl create namespace bookwarehouse
    osm namespace add bookstore bookbuyer bookthief bookwarehouse
    kubectl apply -f https://raw.githubusercontent.com/openservicemesh/osm-docs/release-v1.2/manifests/apps/bookbuyer.yaml
    kubectl apply -f https://raw.githubusercontent.com/openservicemesh/osm-docs/release-v1.2/manifests/apps/bookstore.yaml
    kubectl apply -f https://raw.githubusercontent.com/openservicemesh/osm-docs/release-v1.2/manifests/apps/bookwarehouse.yaml
    kubectl apply -f https://raw.githubusercontent.com/openservicemesh/osm-docs/release-v1.2/manifests/apps/mysql.yaml
done

# ==== Deploy a curl client for testing
${k1} apply -f - <<EOF
apiVersion: v1
kind: Deployment
metadata:
  name: curl
  namespace: bookbuyer
spec:
  replicas: 1
  selector:
    matchLabels:
      app: curl
  template:
    metadata:
      labels:
        app: curl
    spec:
      containers:
      - name: curl
        image: curlimages/curl
        command: ["sleep", "1000000"]
EOF

# ==== 3. Create ServiceExport
${k2} apply -f - <<EOF
apiVersion: multicluster.x-k8s.io/v1alpha1
kind: ServiceExport
metadata:
  name: bookstore
  namespace: bookstore
EOF

# ==== 4. Create ServiceImport

# create a meta-service to get a valid local service IP
${k1} apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: mcs-bookstore
  namespace: bookstore
spec:
  type: ClusterIP
  ports:
  - port: ${app_port}
    protocol: TCP
EOF
sleep 5
mcs_ip=$(${k1} get service mcs-bookstore -n bookstore -o jsonpath='{.spec.clusterIP}')

${k1} apply -f - <<EOF
apiVersion: multicluster.x-k8s.io/v1alpha1
kind: ServiceImport
metadata:
  name: bookstore
  namespace: bookstore
spec:
  ips:
  - ${mcs_ip}
  type: ClusterSetIP
  ports:
  - name: http
    protocol: TCP
    port: ${app_port}
EOF

# ==== 5. Expose service on c2 and get the IP addresses.
${k2} patch service bookstore -n bookstore -p '{"spec":{"type":"LoadBalancer"}}'
for _ in {1..10}; do
    remote_ip=$(${k2} get service bookstore -n bookstore -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

    if [ -n "${remote_ip}" ]; then
        break
    fi
    sleep 5
done
if [ -z "${remote_ip}" ]; then
    echo "Remote bookstore IP not found"
    exit 1
fi
echo "Remote bookstore IP: ${remote_ip}"

local_ip=$(${k1} get service bookstore -n bookstore -o jsonpath='{.spec.clusterIP}')

# ==== 6. Create remote endpointSlice
uid=$(${k1} get serviceimport bookstore -n bookstore -o jsonpath='{.metadata.uid}')
${k1} apply -f - <<EOF
apiVersion: discovery.k8s.io/v1
kind: EndpointSlice
metadata:
  name: imported-bookstore-cluster-c2
  namespace: bookstore
  labels:
    multicluster.kubernetes.io/source-cluster: cluster-c2
    multicluster.kubernetes.io/service-name: bookstore
  ownerReferences:
  - apiVersion: multicluster.k8s.io/v1alpha1
    controller: false
    kind: ServiceImport
    name: bookstore
    uid: ${uid}
addressType: IPv4
ports:
  - name: http
    protocol: TCP
    port: ${app_port}
endpoints:
  - addresses:
    - "${remote_ip}"
EOF

# ==== 7. Update coreDNS entries
${k1} apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: coredns-custom
  namespace: kube-system
data:
  mcs-bookstore.server: |
    bookstore.bookstore.svc.clusterset.local:53 {
      errors
      cache 30
      hosts {
        ${mcs_ip} bookstore.bookstore.svc.clusterset.local
        ${local_ip} bookstore.bookstore.svc.clusterset.local
        fallthrough
      }
    }
EOF

# reload coreDNS
${k1} delete pod --namespace kube-system -l k8s-app=kube-dns

