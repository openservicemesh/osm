---
title: "Ingress Troubleshooting"
description: "Ingress Troubleshooting Guide"
type: docs
aliases: ["ingress.md"]
---

## When Ingress is not working as expected

### 1. Confirm global ingress configuration is set as expected. 

```console
# Returns true if HTTPS ingress is enabled
$ kubectl get ConfigMap osm-config -n osm-system -p jsonpath='{.data.use_https_ingress}{"\n"}'
```

If the output of this command is `false` this means that HTTP ingress is enabled and HTTPS ingress is disabled. To disable HTTP ingress and enable HTTPS ingress, use the following command: 

```bash
# Replace osm-system with osm-controller's namespace if using a non default namespace
kubectl patch ConfigMap osm-config -n osm-system -p '{"data":{"use_https_ingress":"true"}}' --type=merge
```
> Note: Changes made with `kubectl patch` are not preserved across release upgrades. To make this change persistent between upgrades, use `osm mesh upgrade`. See `osm mesh upgrade --help` for more details.


Likewise, to enable HTTP ingress and disable HTTPS ingress, run: 

```bash
# Replace osm-system with osm-controller's namespace if using a non default namespace
kubectl patch ConfigMap osm-config -n osm-system -p '{"data":{"use_https_ingress":"false"}}' --type=merge
```

### 2. Inspect OSM controller logs for errors

```bash
# When osm-controller is deployed in the osm-system namespace
kubectl logs -n osm-system $(kubectl get pod -n osm-system -l app=osm-controller -o jsonpath='{.items[0].metadata.name}')
```

Errors will be logged with the `level` key in the log message set to `error`:
```console
{"level":"error","component":"...","time":"...","file":"...","message":"..."}
```

### 3. Confirm that the ingress resource has been successfully deployed

```bash
kubectl get ingress <ingress-name> -n <ingress-namespace>
```
