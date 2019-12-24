#!/bin/bash

set -aueo pipefail

HOSTS=("bookstore.mesh" "bookstore-1" "bookstore-2")

while true; do

    for HOST in ${HOSTS[@]}; do
        echo -e "---------------------------"
        URL="http://$HOST/"
        echo -e "\ncurl $URL"
        curl -I -s --connect-timeout 1 --max-time 1 $URL || true
    done

    sleep 3
done
