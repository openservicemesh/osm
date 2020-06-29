#!/bin/bash

# shellcheck disable=SC1091
source .env

FILE="./bookstore.pcap"

POD="$(kubectl get pods --selector app=bookstore-v2 -n "$BOOKSTORE_NAMESPACE" --no-headers | grep 'Running' | awk '{print $1}')"

kubectl exec -it -n "$BOOKSTORE_NAMESPACE"  \
        "$POD" \
        -c bookstore-v2 -- \
        tcpdump -A -U -c 1024 -s 0 -w /tmp/pcap.pcap

kubectl cp "$BOOKSTORE_NAMESPACE"/"$POD":/tmp/pcap.pcap "$FILE"
