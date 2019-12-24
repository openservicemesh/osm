#!/bin/bash

set -aueo pipefail

HOST="bookstore.mesh"

while true; do
    echo -e "---------------------------"
    URL="http://$HOST/"
    echo -e "\ncurl $URL"
    curl -I -s --connect-timeout 1 --max-time 1 $URL || true
    sleep 3
done
