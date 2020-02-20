#!/usr/bin/env bash

set -euo pipefail

basedir="$(dirname "$0")/"
webhookCertsDir="$basedir/webhook-certs"
rootCACert="$webhookCertsDir/ca.crt"
name="$1"
namespace="$2"

echo "Creating Kubernetes webhook resoures"

# Read the PEM-encoded CA certificate, base64 encode it, and replace the `${CA_PEM_B64}` placeholder in the YAML
# template with it. Then, create the Kubernetes resources.
ca_pem_b64="$(openssl base64 -A <"${rootCACert}")"
kubectl -n "$namespace" delete mutatingwebhookconfiguration "$name" --ignore-not-found=true
sed -e 's@{{ .CaBundle }}@'"$ca_pem_b64"'@g;s@{{ .Name }}@'"$name"'@g;s@{{ .Namespace }}@'"$namespace"'@g' <"${basedir}/webhook.yaml.template" \
    | kubectl create -f -

rm -rf "$webhookCertsDir"
echo "The webhook resource has been configured!"
