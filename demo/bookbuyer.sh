#!/bin/bash

set -aueo pipefail

HOST="bookstore.mesh"

while true; do
    echo -e "---------------------------"
    URL="http://$HOST/counter"
    echo -e "\ncurl $URL"
    curl -X GET -I -s --connect-timeout 1 --max-time 1 $URL || true
   
    echo -e "---------------------------"
    URL="http://$HOST/incrementcounter"
    echo -e "\ncurl $URL"
    curl -X GET -I -s --connect-timeout 1 --max-time 1 $URL || false
    sleep 3
done
