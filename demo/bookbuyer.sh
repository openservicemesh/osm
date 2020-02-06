#!/bin/bash

set -aueo pipefail

COUNTER="http://bookstore.mesh/counter"
INCREMT="http://bookstore.mesh/incrementcounter"

while true; do
    echo -e "\\n\\n--- $(date) ------------------------"
    echo "curl $COUNTER"
    curl -X GET -I -s --connect-timeout 1 --max-time 1 "$COUNTER" || true
    echo "---"
    echo "curl $INCREMT"
    curl -X GET -I -s --connect-timeout 1 --max-time 1 "$INCREMT" || true
    sleep 3
done
