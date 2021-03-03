#!/bin/bash

# This script will iterate $K8S_NAMESPACE pods and store current logs for all container pods
# and any logs for any previous run of a pod, under <dirPath>/logs/<podname>/<[running|previous]_contname>.log
# If <dirPath> doesn't exist, it is created.
# Requires jq (https://stedolan.github.io/jq/)

# Use: ./get-osm-namespace-logs.sh <dirPath>

# This command will fail on CI as .env does not exist. Can be ignored.
# shellcheck disable=SC1091
source .env > /dev/null 2>&1 

dirName=${1}
mkdir -p "$dirName"

res=$(kubectl get pods -n "$K8S_NAMESPACE" -o json)

# Iterate pod names in K8S_NAMESPACE
for pod in $(echo "$res" |  jq '.items[] | .metadata.name' | sed 's/"//g') ; do
  logStorePath="$dirName/logs/$pod"
  mkdir -p "$logStorePath"

  echo "Checking $pod"

  # For every pod, iterate its containers
  for cont in $(echo "$res" | jq '.items[] | select(.metadata.name|contains("'"$pod"'")) | .spec.containers[].name' | sed 's/"//g') ; do
    # Logs for running instance
    echo "Checking $pod / $cont"
    kubectl logs "$pod" -c "$cont" > "$logStorePath/running_${pod}_${cont}.log"

    # Check if there are logs of previous run (restart)
    kubectl logs "$pod" -c "$cont" -p > /dev/null 2>&1 
    foundPrevLogs=$?
    if [ $foundPrevLogs -eq 0 ]; then
      # There's logs, fetch
      kubectl logs "$pod" -c "$cont" -p > "$logStorePath/previous_${cont}.log"
    fi
  done
done

echo "Done"
exit