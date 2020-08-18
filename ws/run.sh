#!/bin/bash
bin/osm mesh delete -f --mesh-name osm --namespace osm-system
echo "======== deletions done, waiting for sometime ========"
sleep 20

./bin/osm install --container-registry docker.dev.ws:5000 --osm-image-tag latest --enable-permissive-traffic-policy
./bin/osm namespace add default
