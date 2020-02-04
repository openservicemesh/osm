#!/bin/bash

set -aueo pipefail

COUNTER="http://bookstore.mesh/counter"
INCREMT="http://bookstore.mesh/incrementcounter"

while true; do
    echo -e "\n\n--- $(date) ------------------------"
    echo "curl $COUNTER"
    curl -vvv -X GET -I -s --connect-timeout 1 --max-time 1 "$COUNTER" || true
    echo "exit code: $?"
    echo "---"
    echo "curl $INCREMT"
    curl -vvv -X GET -I -s --connect-timeout 1 --max-time 1 "$INCREMT" || true
    echo "exit code: $?"
    sleep 3
done
