#!/bin/bash

POD_CIDR=$(kubectl cluster-info dump | grep -m 1 -Po '(?<=--cluster-cidr=)[0-9.\/]+')

# Apply a spec with invalid IP to figure out the valid IP range
SERVICE_CIDR=$(echo '{"apiVersion":"v1","kind":"Service","metadata":{"name":"fake"},"spec":{"clusterIP":"1.1.1.1","ports":[{"port":443}]}}' | kubectl apply -f - 2>&1 | sed 's/.*valid IPs is //')

echo "$POD_CIDR,$SERVICE_CIDR"
