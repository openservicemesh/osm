#!/bin/bash


# This script captures certain amount of network traffic from the BOOKSTORE V1 pod.
# The capture is then moved from pod to local host.
# This is a pcap file that can be inspected with tools like Wireshark.
# This is useful to show encrypted/unencrypted traffic (demo enabling mTLS).


# shellcheck disable=SC1091
source .env

POD="$(kubectl get pods --selector app=bookstore-v1 -n "$BOOKSTORE_NAMESPACE" --no-headers | grep 'Running' | awk '{print $1}')"


kubectl exec -it -n "$BOOKSTORE_NAMESPACE"  \
        "$POD" \
        -c "bookstore-v1" -- \
        tcpdump -A -U -c 1024 -s 0 -w /tmp/pcap.pcap


# Move the file from pod to local host
kubectl cp "$BOOKSTORE_NAMESPACE"/"$POD":/tmp/pcap.pcap ./
