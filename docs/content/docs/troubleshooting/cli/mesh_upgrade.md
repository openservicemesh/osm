---
title: "OSM Mesh Upgrade Command Troubleshooting"
description: "OSM Mesh Upgrade Command Troubleshooting Guide"
type: docs
---

## OSM Mesh Upgrade Timing Out
### Insufficient CPU
If the `osm mesh upgrade` command is timing out, it could be due to insufficient CPU.
1. Check the pods to see if any of them aren't fully up and running
```bash
# Replace osm-system with osm-controller's namespace if using a non-default namespace
kubectl get pods -n osm-system
```
2. If there are any pods that are in Pending state, use `kubectl describe` to check the `Events` section
```bash
# Replace osm-system with osm-controller's namespace if using a non-default namespace
kubectl describe pod <pod-name> -n osm-system
```

If you see the following error, then please increase the number of CPUs Docker can use.
```bash
`Warning  FailedScheduling  4s (x15 over 19m)  default-scheduler  0/1 nodes are available: 1 Insufficient cpu.`
```
### Error Validating CLI Parameters
If the `osm mesh upgrade` command is still timing out, it could be due to a CLI/Image Version mismatch.

1. Check the pods to see if any of them aren't fully up and running
```bash
# Replace osm-system with osm-controller's namespace if using a non-default namespace
kubectl get pods -n osm-system
```
2. If there are any pods that are in Pending state, use `kubectl describe` to check the `Events` section for `Error Validating CLI parameters`
```bash
# Replace osm-system with osm-controller's namespace if using a non-default namespace
kubectl describe pod <pod-name> -n osm-system
```
3. If you find the error, please check the pod's logs for any errors
```bash
kubectl logs -n osm-system <pod-name> | grep -i error
```

If you see the following error, then it's due to a CLI/Image Version mismatch.
```bash
`"error":"Please specify the init container image using --init-container-image","reason":"FatalInvalidCLIParameters"`
```
Workaround is to set the `container-registry` and `osm-image-tag` flag when running `osm mesh upgrade`.
```bash
osm mesh upgrade --container-regsitry $CTR_REGISTRY --osm-image-tag $CTR_TAG --enable-egress=true
```

## Other Issues
If you're running into issues that are not resolved with the steps above, please [open a GitHub issue](https://github.com/openservicemesh/osm/issues).
