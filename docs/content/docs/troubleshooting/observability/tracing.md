---
title: "Troubleshoot Tracing/Jaeger"
description: "How to fix common issues with OSM's tracing integration"
type: docs
---

# When tracing is not working as expected

## 1. Errors enabling tracing
If you face issues with using `osm mesh upgrade` to enable tracing on a running OSM instance, troubleshoot [here](https://docs.openservicemesh.io/docs/troubleshooting/cli/mesh_upgrade/)

## 2. Verify that tracing is enabled
Ensure the `tracing_enable` key in the `osm-config` ConfigMap is set to `true`:
```bash
kubectl get configmap -n osm-system osm-config -o json | jq '.data.tracing_enable'
"true"
```

## 3. Verify the tracing values being set are as expected 
If tracing is enabled, you can verify the specific `tracing_address`, `tracing_port` and `tracing_endpoint` being used for tracing in the ConfigMap:
```bash
kubectl get configmap osm-config -n osm-system -o json | jq '.data'
```
To verify that the Envoys point to the FQDN you intend to use, check the `tracing_address` value.

## 4. Verify the tracing values being used are as expected
To dig one level deeper, you may also check whether the values set by the ConfigMap are being correctly used. Use the command below to get the config dump of the pod in question and save the output in a file.
```bash
osm proxy get config_dump -n <pod-namespace> <pod-name> > <file-name>
```
Open the file in your favorite text editor and search for `envoy-tracing-cluster`. You should be able to see the tracing values in use. Example output for the bookbuyer pod:
```json
"name": "envoy-tracing-cluster",
      "type": "LOGICAL_DNS",
      "connect_timeout": "1s",
      "alt_stat_name": "envoy-tracing-cluster",
      "load_assignment": {
       "cluster_name": "envoy-tracing-cluster",
       "endpoints": [
        {
         "lb_endpoints": [
          {
           "endpoint": {
            "address": {
             "socket_address": {
              "address": "jaeger.osm-system.svc.cluster.local",
              "port_value": 9411
        [...]
```

## 5. Verify that the OSM Controller was installed with Jaeger automatically deployed [optional]
If you used automatic bring-up, you can additionally check for the Jaeger service and Jaeger deployment:
```bash
# Assuming OSM is installed in the osm-system namespace:
kubectl get services -n osm-system -l app=jaeger

NAME     TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)    AGE
jaeger   ClusterIP   10.99.2.87   <none>        9411/TCP   27m
```

```bash
# Assuming OSM is installed in the osm-system namespace:
kubectl get deployments -n osm-system -l app=jaeger

NAME     READY   UP-TO-DATE   AVAILABLE   AGE
jaeger   1/1     1            1           27m
```

## 6. Verify Jaeger pod readiness, responsiveness and health
Check if the Jaeger pod is running in the namespace you have deployed it in:
> The commands below are specific to OSM's automatic deployment of Jaeger; substitute namespace and label values for your own tracing instance as applicable:
```bash
kubectl get pods -n osm-system -l app=jaeger

NAME                     READY   STATUS    RESTARTS   AGE
jaeger-8ddcc47d9-q7tgg   1/1     Running   5          27m
```

To get information about the Jaeger instance, use `kubectl describe pod` and check the `Events` in the output.
```bash
kubectl describe pod -n osm-system -l app=jaeger
```

## External Resources
* [Jaeger Troubleshooting docs](https://www.jaegertracing.io/docs/1.22/troubleshooting/)
