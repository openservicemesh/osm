---
title: "Iptables Redirection"
description: "Iptables Redirection"
type: docs
aliases: ["iptables_redirection.md"]
---

# Iptables Redirection

OSM leverages [iptables](https://linux.die.net/man/8/iptables) to intercept and redirect traffic to and from pods participating in the service mesh to the Envoy proxy sidecar container running on each pod. Traffic redirected to the Envoy proxy sidecar is filtered and routed based on service mesh traffic policies.

## How it works

OSM sidecar injector service `osm-injector` injects an Envoy proxy sidecar on every pod created within the service mesh. Along with the Envoy proxy sidecar, `osm-injector` also injects an [init container](https://kubernetes.io/docs/concepts/workloads/pods/init-containers/), a specialized container that runs before any application containers in a pod. The injected init container is responsible for bootstrapping the application pods with traffic redirection rules such that all outbound TCP traffic from a pod and all inbound traffic TCP traffic to a pod are redirected to the envoy proxy sidecar running on that pod. This redirection is set up by the init container by running a set of `iptables` commands.

### Ports reserved for traffic redirection

OSM reserves a set of port numbers to perform traffic redirection and provide admin access to the Envoy proxy sidecar. It is essential to note that these port numbers must not be used by application containers running in the mesh. Using any of these reserved port numbers will lead to the Envoy proxy sidecar not functioning correctly.

Following are the port numbers that are reserved for use by OSM:

1. `15000`: used by the [Envoy admin interface](https://www.envoyproxy.io/docs/envoy/latest/operations/admin) exposed over `localhost`
1. `15001`: used by the Envoy outbound listener to accept and proxy outbound traffic sent by applications within the pod
1. `15003`: used by the Envoy inbound listener to accept and proxy inbound traffic entering the pod destined to applications within the pod
1. `15010`: used by the Envoy inbound Prometheus listener to accept and proxy inbound traffic pertaining to scraping Envoy's Prometheus metrics
1. `15901`: used by Envoy to serve rewritten HTTP liveness probes
1. `15902`: used by Envoy to serve rewritten HTTP readiness probes
1. `15903`: used by Envoy to serve rewritten HTTP startup probes

### Application User ID (UID) reserved for traffic redirection

OSM reserves the user ID (UID) value `1500` for the Envoy proxy sidecar container. This user ID is of utmost importance while performing traffic interception and redirection to ensure the redirection does not result in a loop. The user ID value `1500` is used to program redirection rules to ensure redirected traffic from Envoy is not redirected back to itself!

Application containers must not used the reserved user ID value of `1500`.

### Types of traffic intercepted

Currently, OSM programs the Envoy proxy sidecar on each pod to only intercept inbound and outbound `TCP` traffic. This includes raw `TCP` traffic and any application traffic that uses `TCP` as the underlying transport protocol, such as `HTTP`, `gRPC` etc. This implies `UDP` and `ICMP` traffic which can be intercepted by `iptables` are not intercepted and redirected to the Envoy proxy sidecar.

### Iptables chains and rules

OSM's `osm-injector` service programs the init container to set up a set of `iptables` chains and rules to perform traffic interception and redirection. The following section provides details on the responsibility of these chains and rules.

OSM leverages four chains to perform traffic interception and redirection:

1. `PROXY_INBOUND`: chain to intercept inbound traffic entering the pod
1. `PROXY_IN_REDIRECT`: chain to redirect intercepted inbound traffic to the sidecar proxy's inbound listener
1. `PROXY_OUTPUT`: chain to intercept outbound traffic from applications within the pod
1. `PROXY_REDIRECT`: chain to redirect intercepted outbound traffic to the sidecar proxy's outbound listener

Each of the chains above are programmed with rules to intercept and redirect application traffic via the Envoy proxy sidecar.

### Global outbound IP range exclusions

Outbound TCP based traffic from applications is by default intercepted using the `iptables` rules programmed by OSM, and redirected to the Envoy proxy sidecar. In some cases, it might be desirable to not subject certain IP ranges to be redirected and routed by the Envoy proxy sidecar based on service mesh policies. A common use case to exclude IP ranges is to not route non-application logic based traffic via the Envoy proxy, such as traffic destined to the Kubernetes API server, or traffic destined to a cloud provider's instance metadata service. In such scenarios, excluding certain IP ranges from being subject to service mesh traffic routing policies becomes necessary.

OSM provides a means to specify a global list of IP ranges to exclude from outbound traffic interception in the following ways:

1. During OSM install using the `--set` option:
    ```bash
    # To exclude the IP ranges 1.1.1.1/32 and 2.2.2.2/24 from outbound interception
    osm install --set="OpenServiceMesh.outboundIPRangeExclusionList={1.1.1.1/32,2.2.2.2/24}
    ```

1. By setting the `outbound_ip_range_exclusion_list` key in the `osm-config` ConfigMap:
    ```bash
    ## Assumes OSM is installed in the osm-system namespace
    kubectl patch ConfigMap osm-config -n osm-system -p '{"data":{"outbound_ip_range_exclusion_list":"1.1.1.1/32, 2.2.2.2/24"}}' --type=merge
    ```

1. By uppgrading the Helm chart directly if OSM was installed using Helm:
    ```bash
    osm mesh upgrade --outbound-ip-range-exclusion-list "1.1.1.1/32,2.2.2.2/24"
    ```

Excluded IP ranges are stored in the `osm-config` ConfigMap with the key `outbound_ip_range_exclusion_list`, and is read at the time of sidecar injection by `osm-injector`. These dynamically configurable IP ranges are programmed by the init container along with the static rules used to intercept and redirect traffic via the Envoy proxy sidecar. Excluded IP ranges will not be intercepted for traffic redirection to the Envoy proxy sidecar.

## Sample demo

### Traffic redirection with IP range exclusions

The following demo shows an HTTP `curl` client making HTTP requests to the `httpbin.org` website directly using its IP address. We will explicitly disable the egress functionality to ensure traffic to a non-mesh destination (`httpbin.org` in this demo) is not able to egress the pod.

1. Install OSM with egress disabled.
    ```bash
    osm install --enable-egress=false
    ```

1. Deploy the `curl` client into the `curl` namespace after enrolling its namespace to the mesh.

    ```bash
    # Create the curl namespace
    kubectl create namespace curl

    # Add the namespace to the mesh
    osm namespace add curl

    # Deploy curl client in the curl namespace
    kubectl apply -f docs/example/manifests/samples/curl/curl.yaml -n curl
    ```

    Confirm the `curl` client pod is up and running.

    ```console
    $ kubectl get pods -n curl
    NAME                    READY   STATUS    RESTARTS   AGE
    curl-54ccc6954c-9rlvp   2/2     Running   0          20s
    ```

1. Retrieve the public IP address for the `httpbin.org` website. For the purpose of this demo, we will test with a single IP range to be excluded from traffic interception. In this example, we will use the IP address `54.91.118.50` represented by the IP range `54.91.118.50/32`, to make HTTP requests with and without outbound IP range exclusions configured.
    ```console
    $ nslookup httpbin.org
    Server:		172.23.48.1
    Address:	172.23.48.1#53

    Non-authoritative answer:
    Name:	httpbin.org
    Address: 54.91.118.50
    Name:	httpbin.org
    Address: 54.166.163.67
    Name:	httpbin.org
    Address: 34.231.30.52
    Name:	httpbin.org
    Address: 34.199.75.4
    ```

1. Confirm the `curl` client is unable to make successful HTTP requests to the `httpbin.org` website running on `http://54.91.118.50:80`.

    ```console
    $ kubectl exec -n curl -ti "$(kubectl get pod -n curl -l app=curl -o jsonpath='{.items[0].metadata.name}')" -c curl -- curl -I http://54.91.118.50:80
    curl: (7) Failed to connect to 54.91.118.50 port 80: Connection refused
    command terminated with exit code 7
    ```

    The failure above is expected because by default outbound traffic is redirected via the Envoy proxy sidecar running on the `curl` client's pod, and the proxy subjects this traffic to service mesh policies which does not allow this traffic.

1. Program OSM to exclude the IP range `54.91.118.50/32` IP range
    ```bash
    ## Assumes OSM is installed in the osm-system namespace
    kubectl patch ConfigMap osm-config -n osm-system -p '{"data":{"outbound_ip_range_exclusion_list":"54.91.118.50/32"}}' --type=merge
    ```

1. Confirm the ConfigMap has been updated as expected
    ```console
    # 54.91.118.50 is one of the IP addresses for httpbin.org
    $ kubectl get configmap -n osm-system osm-config -o jsonpath='{.data.outbound_ip_range_exclusion_list}{"\n"}'
    54.91.118.50/32
    ```

1. Restart the `curl` client pod so the updated outbound IP range exclusions can be configured. It is important to note that existing pods must be restarted to pick up the updated configuration because the traffic interception rules are programmed by the init container only at the time of pod creation.
    ```bash
    kubectl rollout restart deployment curl -n curl
    ```

    Wait for the restarted pod to be up and running.

1. Confirm the `curl` client is able to make successful HTTP requests to the `httpbin.org` website running on `http://54.91.118.50:80`
    ```console
    # 54.91.118.50 is one of the IP addresses for httpbin.org
    $ kubectl exec -n curl -ti "$(kubectl get pod -n curl -l app=curl -o jsonpath='{.items[0].metadata.name}')" -c curl -- curl -I http://54.91.118.50:80
    HTTP/1.1 200 OK
    Date: Thu, 18 Mar 2021 23:17:44 GMT
    Content-Type: text/html; charset=utf-8
    Content-Length: 9593
    Connection: keep-alive
    Server: gunicorn/19.9.0
    Access-Control-Allow-Origin: *
    Access-Control-Allow-Credentials: true
    ```

1. Confirm that HTTP requests to other IP addresses of the `httpbin.org` website that are not excluded fail
    ```console
    # 34.199.75.4 is one of the IP addresses for httpbin.org
    $ kubectl exec -n curl -ti "$(kubectl get pod -n curl -l app=curl -o jsonpath='{.items[0].metadata.name}')" -c curl -- curl -I http://34.199.75.4:80
    curl: (7) Failed to connect to 34.199.75.4 port 80: Connection refused
    command terminated with exit code 7
    ```

## Iptables configuration

Iptables rules are programmed by OSM's init container when a pod is created in the mesh. The rules are on the pod via a set of `iptables` commands run by the init container.

The following snippet from the demo `curl` client's init container spec shows the set of `iptables` commands along with exclusion rules for reference.

```console
Init Containers:
  osm-init:
    Container ID:  containerd://80f86af7bc64b7a70f7f2bf64242d735d857559a79cd97e206513368130902f1
    Image:         openservicemesh/init:v0.8.0
    Image ID:      docker.io/openservicemesh/init@sha256:eb1f6ab02aeaaba8f58aaa29406b1653d7a3983958ea040c2af8845136ed786c
    Port:          <none>
    Host Port:     <none>
    Command:
      /bin/sh
    Args:
      -c
      iptables -t nat -N PROXY_INBOUND && iptables -t nat -N PROXY_IN_REDIRECT && iptables -t nat -N PROXY_OUTPUT && iptables -t nat -N PROXY_REDIRECT && iptables -t nat -A PROXY_REDIRECT -p tcp -j REDIRECT --to-port 15001 && iptables -t nat -A PROXY_REDIRECT -p tcp --dport 15000 -j ACCEPT && iptables -t nat -A OUTPUT -p tcp -j PROXY_OUTPUT && iptables -t nat -A PROXY_OUTPUT -m owner --uid-owner 1500 -j RETURN && iptables -t nat -A PROXY_OUTPUT -d 127.0.0.1/32 -j RETURN && iptables -t nat -A PROXY_OUTPUT -j PROXY_REDIRECT && iptables -t nat -A PROXY_IN_REDIRECT -p tcp -j REDIRECT --to-port 15003 && iptables -t nat -A PREROUTING -p tcp -j PROXY_INBOUND && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15010 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15901 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15902 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp --dport 15903 -j RETURN && iptables -t nat -A PROXY_INBOUND -p tcp -j PROXY_IN_REDIRECT && iptables -t nat -I PROXY_OUTPUT -d 54.91.118.50/32 -j RETURN
    State:          Terminated
      Reason:       Completed
      Exit Code:    0
      Started:      Thu, 18 Mar 2021 16:14:30 -0700
      Finished:     Thu, 18 Mar 2021 16:14:30 -0700
    Ready:          True
    Restart Count:  0
    Environment:    <none>
    Mounts:
      /var/run/secrets/kubernetes.io/serviceaccount from curl-token-c4jv9 (ro)
```