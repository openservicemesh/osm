#!/bin/bash

set -aueo pipefail

while true; do

    for HOST in "bookstore.mesh"; do
        echo -e "---------------------------"
        URL="http://$HOST/"
        echo -e "\ncurl $URL"
        curl -I -s --connect-timeout 1 --max-time 1 $URL || true
    done

    sleep 3
done
