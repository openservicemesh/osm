#!/usr/bin/env bash

set -euo pipefail

basedir="$(dirname "$0")/"
webhookCertsDir="$basedir/webhook-certs"
name="$1"
namespace="$2"
instanceID="$3"
webhookname="${namespace}-${name}-webhook"
echo "Creating Kubernetes webhook resoures"

CA_BUNDLE_SECRET="osm-ca-${instanceID}"

CA=$(kubectl get secrets "$CA_BUNDLE_SECRET" -n "$namespace" -o yaml | grep 'ca.crt' | awk '{print $2}')

kubectl -n "$namespace" delete mutatingwebhookconfiguration "$name" --ignore-not-found=true
sed -e 's@{{ .CaBundle }}@'"$CA"'@g;s@{{ .WebhookName }}@'"$webhookname"'@g;s@{{ .Name }}@'"$name"'@g;s@{{ .Namespace }}@'"$namespace"'@g;s@{{ .InstanceID }}@'"$instanceID"'@g' <"${basedir}/webhook.yaml.template" \
    | kubectl create -f -

rm -rf "$webhookCertsDir"
echo "The webhook resource has been configured!"
