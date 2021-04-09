---
title: "Tracing"
description: "Tracing with Jaeger"
type: docs
aliases: ["tracing.md"]
---

# Tracing
Open Service Mesh (OSM) allows optional deployment of Jaeger for tracing. Similarly, tracing can be enabled and customized during installation (`tracing` section in `values.yaml`) or at runtime by editing the `osm-config` ConfigMap. Tracing can be enabled, disabled and configured at any time to support BYO scenarios.

When OSM is deployed with tracing enabled, the OSM control plane will use the [user-provided tracing information](#tracing-values) to direct the Envoys to send traces when and where appropriate. If tracing is enabled without user-provided values, it will use the defaults in `values.yaml`. The `tracing-address` value tells all Envoys injected by OSM the FQDN to send tracing information to.

OSM supports tracing with applications that use Zipkin protocol.

## Jaeger
[Jaeger](https://www.jaegertracing.io/) is an open source distributed tracing system used for monitoring and troubleshooting distributed systems. It allows you to get fine-grained metrics and distributed tracing information across your setup so that you can observe which microservices are communicating, where requests are going, and how long they are taking. You can use it to inspect for specific requests and responses to see how and when they happen.

When tracing is enabled, Jaeger is capable of receiving spans from Envoys in the mesh that can then be viewed and queried on Jaeger's UI via port-forwarding.

OSM CLI offers the capability to deploy a Jaeger instance with OSM's installation, but bringing your own managed Jaeger and configuring OSM's tracing to point to it later is also supported. 

### Automatically Provision Jaeger
By default, Jaeger deployment and tracing as a whole is disabled.

A Jaeger instance can be automatically deployed by using the `--deploy-jaeger` OSM CLI flag at install time. This will provision a Jaeger pod in the mesh namespace.

Additionally, OSM has to be instructed to enable tracing on the proxies; this is done via the `tracing` section on the ConfigMap.

The following command will both deploy Jaeger and configure the tracing parameters according to the address of the newly deployed instance of Jaeger during OSM installation:
```bash
osm install --deploy-jaeger --set OpenServiceMesh.tracing.enable=true
```

This default bring-up uses the [All-in-one Jaeger executable](https://www.jaegertracing.io/docs/1.22/getting-started/#all-in-one) that launches the Jaeger UI, collector, query, and agent. 

### BYO (Bring-your-own)
This section documents the additional steps needed to allow an already running instance of Jaeger to integrate with your OSM control plane.
> NOTE: This guide outlines steps specifically for Jaeger but you may use your own tracing application instance with applicable values. OSM supports tracing with applications that use Zipkin protocol

#### Prerequisites
* A running Jaeger instance
    * [Getting started with Jaeger](https://www.jaegertracing.io/docs/1.22/getting-started/) includes a sample app as a demo

#### Tracing Values
The sections below outline how to make required updates depending on whether you already already have OSM installed or are deploying tracing and Jaeger during OSM installation. In either case, the following `tracing` values in `values.yaml` are being updated to point to your Jaeger instance:
1. `enable`: set to `true` to tell the Envoy connection manager to send tracing data to a specific address (cluster)
1. `address`: set to the destination cluster of your Jaeger instance
1. `port`: set to the destination port for the listener that you intend to use
1. `endpoint`: set to the destination's API or collector endpoint where the spans will be sent to


#### a) Enable tracing after OSM control plane has already been installed

If you already have OSM running, `tracing` values must be updated in the OSM ConfigMap using:

```bash
osm mesh upgrade --enable-tracing --tracing-address <tracing server hostname> --tracing-port <tracing server port> --tracing-endpoint <tracing server endpoint>
```
> NOTE: This command upgrades an OSM control plane configuration by upgrading the underlying Helm release. Note that if you choose to use `kubectl patch` instead of using the OSM CLI, your change will not be preserved across release upgrades. 

If you face issues with using `osm mesh upgrade`, troubleshoot [here](https://docs.openservicemesh.io/docs/troubleshooting/cli/mesh_upgrade/).

You can verify these changes have been deployed by inspecting data values in `osm-config`:
```bash
kubectl get configmap osm-config -n osm-system -o json | jq '.data'
```

#### b) Enable tracing at OSM control plane install time

To deploy your own instance of Jaeger during OSM installation, you can use the `--set` flag as shown below to update the values:

```bash
osm install --set OpenServiceMesh.tracing.enable=true,OpenServiceMesh.tracing.address=<tracing server hostname>,OpenServiceMesh.tracing.port=<tracing server port>,OpenServiceMesh.tracing.endpoint=<tracing server endpoint>
```

## View the Jaeger UI with Port-Forwarding
Jaeger's UI is running on port 16686. To view the web UI, you can use `kubectl port-forward`:

```bash
OSM_POD=$(kubectl get pods -n "$K8S_NAMESPACE" --no-headers  --selector app=jaeger | awk 'NR==1{print $1}')

kubectl port-forward -n "$K8S_NAMESPACE" "$OSM_POD"  16686:16686
```
Navigate to `http://localhost:16686/` in a web browser to view the UI.


## Example of Tracing with Jaeger
This section walks through the process of creating a simple Jaeger instance and enabling tracing with Jaeger in OSM.

1. Run the [OSM Demo](https://github.com/openservicemesh/osm/blob/release-v0.8/demo/README.md) with Jaeger deployed. You have two options:
    - For automatic provisioning of Jaeger, simply set `DEPLOY_JAEGER` in your `.env` file to true
    - For bring-your-own, you can deploy the sample instance [provided by Jaeger](https://www.jaegertracing.io/docs/1.22/getting-started/#all-in-one) using the commands below. If you wish to bring up Jaeger in a different namespace, make sure to update it below.

        Create the Jaeger service.
        ```yaml
        kubectl apply -f - <<EOF
        ---
        kind: Service
        apiVersion: v1
        metadata:
          name: jaeger
          namespace: osm-system
          labels:
            app: jaeger
        spec:
          selector:
            app: jaeger
          ports:
          - protocol: TCP
            # Service port and target port are the same
            port: 9411
          type: ClusterIP
        EOF
        ```        
        
        Create the Jaeger deployment.
        ```yaml
        kubectl apply -f - <<EOF
        ---
        apiVersion: apps/v1
        kind: Deployment
        metadata:
          name: jaeger
          namespace: osm-system
          labels:
            app: jaeger
        spec:
          replicas: 1
          selector:
            matchLabels:
              app: jaeger
          template:
            metadata:
              labels:
                app: jaeger
            spec:
              containers:
              - name: jaeger
                image: jaegertracing/all-in-one
                args:
                  - --collector.zipkin.host-port=9411
                imagePullPolicy: IfNotPresent
                ports:
                - containerPort: 9411
                resources:
                  limits:
                    cpu: 500m
                    memory: 512M
                  requests:
                    cpu: 100m
                    memory: 256M
        EOF
        ```

1. Enable tracing and pass in applicable values. If you have installed Jaeger in a different namespace, replace `osm-system` below.

    ```bash
    osm mesh upgrade --enable-tracing --tracing-address jaeger.osm-system.svc.cluster.local --tracing-port 9411 --tracing-endpoint /api/v2/spans
    ```

1. Refer to instructions [above](#view-the-jaeger-ui-with-port-forwarding) to view the web UI using port forwarding

1. In the browser, you should see a `Service` dropdown which allows you to select from the various applications deployed by the bookstore demo.
    
    a) Select a service to view all spans from it. For example, if you select `bookbuyer` with a `Lookback` of one hour, you can see its interactions with `bookstore-v1` and `bookstore-v2` sorted by time.
    <p align="center">
        <img src="../../images/jaeger-search-traces.png" width="100%"/>
    </p>
    <center><i>Jaeger UI search for bookbuyer traces</i></center><br>

    b) Click on any item to view it in further detail
    
    c) Select multiple items to compare traces. For example, you can compare the `bookbuyer`'s interactions with `bookstore-v1` and `bookstore-v2` at a particular moment in time:
    <p align="center">
        <img src="../../images/jaeger-compare-traces.png" width="100%"/>
    </p>
    <center><i>bookbuyer interactions with bookstore-v1 and bookestore-v2</i></center><br>

    d) Click on the `System Architecture` tab to view a graph of how the various applications have been interacting/communicating. This provides an idea of how traffic is flowing between the applications.
    <p align="center">
        <img src="../../images/jaeger-system-architecture.png" width="40%"/>
    </p>
    <center><i>Directed acyclic graph of bookstore demo application interactions</i></center><br>

If you are not seeing the bookstore demo applications in the Jaeger UI, tail the `bookbuyer` logs to ensure that the applications are successfully interacting.

```bash
POD="$(kubectl get pods -n "$BOOKBUYER_NAMESPACE" --show-labels --selector app=bookbuyer --no-headers | grep -v 'Terminating' | awk '{print $1}' | head -n1)"

kubectl logs "${POD}" -n "$BOOKBUYER_NAMESPACE" -c bookbuyer --tail=100 -f
```

Expect to see:
```bash
"MAESTRO! THIS TEST SUCCEEDED!"
```
This suggests that the issue is not caused by your Jaeger or tracing configuration.

## Troubleshoot
To troubleshoot issues with Jaeger and tracing, check out the [tracing troubleshooting guide](https://docs.openservicemesh.io/docs/troubleshooting/observability/tracing).
