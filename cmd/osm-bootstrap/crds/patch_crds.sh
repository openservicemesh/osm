#!/bin/bash

set -x
set -e

cd bin

CRD_LIST="meshconfigs.config.openservicemesh.io egresses.policy.openservicemesh.io ingressbackends.policy.openservicemesh.io httproutegroups.specs.smi-spec.io httproutegroups.specs.smi-spec.io tcproutes.specs.smi-spec.io traffictargets.access.smi-spec.io trafficsplits.split.smi-spec.io"

for CRD in $CRD_LIST
do
    # First ensure that crd exists
    get_crd=$(./kubectl get crd "$CRD" --ignore-not-found)
    if [ "$get_crd" != "" ]; then
        # Patch the crd conversion spec by setting the strategy to `None` if it isn't already set to `None`
        conv_strategy=$(kubectl get crd "$CRD" -n osm-system -o jsonpath='{.spec.conversion.strategy}{"\n"}')
        if [ "$conv_strategy" != "None" ]; then
            ./kubectl patch crd "$CRD" --type='json' -p '[{"op" : "remove", "path" : "/spec/conversion/webhook"},{"op":"replace", "path":"/spec/conversion/strategy","value" : "None"}]'
        fi
    fi
done

./kubectl apply -f /osm-crds
