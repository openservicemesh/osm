#!/bin/bash

set -aueo pipefail

# shellcheck disable=SC1091
source .env

CURL_NAMESPACE="${CURL_NAMESPACE:-curl}"
curlpod=$(kubectl get pods -n "$CURL_NAMESPACE" -o name | head -1)

if [ -z "$curlpod" ]; then
  echo "No curl pods exist in namespace $CURL_NAMESPACE"
  exit 1
fi

times=10000
echo "Measuring the time it takes to curl $times times from the curl app to the hello app"

curlcmd="for i in \`seq 1 $times\`; do curl -w '%{time_namelookup} %{time_connect} %{time_appconnect} %{time_pretransfer} %{time_starttransfer} %{time_total}\n' -s http://hello.hello.svc.cluster.local:80 -o /dev/null; done"
timing_results=$(kubectl -n curl exec -it "$curlpod" -c curl -- sh -c "$curlcmd")

echo -n "total sum of time_namelookup: "; echo "$timing_results" | awk '{ time_namelookup+=$1 } END { print time_namelookup }'
echo -n "total sum of time_connect: "; echo "$timing_results" | awk '{ time_connect+=$2 } END { print time_connect }'
echo -n "total sum of time_appconnect: "; echo "$timing_results" | awk '{ time_appconnect+=$3 } END { print time_appconnect }'
echo -n "total sum of time_pretransfer: "; echo "$timing_results" | awk '{ time_pretransfer+=$4 } END { print time_pretransfer }'
echo -n "total sum of time_starttransfer: "; echo "$timing_results" | awk '{ time_starttransfer+=$5 } END { print time_starttransfer }'
echo -n "total sum of time_total: "; echo "$timing_results" | awk '{ time_total+=$6 } END { print time_total }'
