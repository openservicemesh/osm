# OPA-plugin-enabled OSM POC

## Overview and limitations
- Allows configuring an envoy's [External Authorization extension](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/ext_authz_filter) through OSM's configmap.
- Authorization filter is currently applied in inmesh `inbound` and `ingress` connections. 
- This demo DOES NOT inject the OPA plugin sidecar on every pod, though this seems to be the intended model for OPA to run with for obvious latency reasons.
  
If using `values.yaml`:
 ```
  # External authz
  inbound_extauthz:
    enable: true
    address: opa.opa.svc.cluster.local
    port: 9191
    statPrefix: authz_opa
    timeout: 1s
    failureModeAllow: false
```
Example OSM's configmap deployed by this POC:
```
  inbound_extauthz_enable: true
  inbound_extauthz_address: opa.opa.svc.cluster.local
  inbound_extauthz_port: 9191
  inbound_extauthz_statprefix: authz_opa
  inbound_extauthz_timeout: 1s
  inbound_extauthz_failuremodeallow: false
```

- Uses (as `opa-envoy-plugin`) a configmap to set OPA's policy. [This is not intended for production](https://github.com/open-policy-agent/opa-envoy-plugin#example-bundle-configuration). 
- `opa-envoy-plugin` does not seem to react to changes on the configmap. To update policies, currently the OPA container needs to be restarted.

## Demo Walkthrough

- Deploy an `opa-envoy-plugin`, use curated yaml in this folder.
  - NOTE: This POC is using a single plugin to handle all pod connections, and through the network. This is not intended for production.
```
kubectl create namespace opa
kubectl apply -f docs/example/manifests/opa/deploy-opa-envoy.yaml
```
`deploy-opa-envoy.yaml` will deploy OPA's envoy plugin and run it as a service, allowing external authorization calls from envoys through the network.

- Deploy OSM's Demo, follow `demo/run-osm-demo.sh`
```
demo/run-osm-demo.sh  # wait for all services to come up
```
- This branch has an External Authorization server expected by default at `opa.opa.svc.cluster.local:9191`. Timeout for authorization RTT is `1s` by default, and will not allow traffic in case of failure.

Traffic should fail right out of the bat:
```
kubectl logs <bookbuyer_pod> -n bookbuyer bookbuyer
```
```
...
--- bookbuyer:[ 8 ] -----------------------------------------

Fetching http://bookstore.bookstore:14001/books-bought
Request Headers: map[Client-App:[bookbuyer] User-Agent:[Go-http-client/1.1]]
Identity: n/a
Booksbought: n/a
Server: envoy
Date: Tue, 04 May 2021 01:20:39 GMT
Status: 403 Forbidden
ERROR: response code for "http://bookstore.bookstore:14001/books-bought" is 403;  expected 200
...
```

You shoud also be able to see the logs in `opa-envoy-plugin` for rejected authorization calls:
```
kubectl logs <opa_pod> -n opa
```
```
{"decision_id":"1df154b5-658a-47bf-ac18-be52998605da","input":{"attributes":{"destination":{"address":{"socketAddress":{"address":"10.0.16.44","portValue":14001}}},"metadataContext":{},"request":{"http":{"headers":{":authority":"bookstore.bookstore:14001",":method":"GET",":path":"/books-bought","accept-encoding":"gzip","client-app":"bookbuyer","user-agent":"Go-http-client/1.1","x-forwarded-proto":"http","x-request-id":"69b80716-6af4-4986-bf8c-8f209d96f131"},"host":"bookstore.bookstore:14001","id":"6079090369556950701","method":"GET","path":"/books-bought","protocol":"HTTP/1.1"},"time":"2021-05-04T01:21:18.195876Z"},"source":{"address":{"socketAddress":{"address":"10.244.2.10","portValue":53488}}}},"parsed_body":null,"parsed_path":["books-bought"],"parsed_query":{},"truncated_body":false,"version":{"encoding":"protojson","ext_authz":"v3"}},"labels":{"id":"6e56bc11-a212-4c3e-be4d-b33186fd581d","version":"0.28.0-envoy"},"level":"info","metrics":{"timer_rego_query_eval_ns":105799,"timer_server_handler_ns":425097},"msg":"Decision Log","path":"envoy/authz/allow","requested_by":"","result":false,"time":"2021-05-04T01:21:18Z","timestamp":"2021-05-04T01:21:18.1971808Z","type":"openpolicyagent.org/decision_logs"}
```

- Now edit OPA's policy:
```
kubectl edit configmap opa-policy -n opa
```
change specifically the default-all from:
```
default allow = false
```
to
```
default allow = true
```

- Finally, restart `opa-envoy-plugin`:
```
kubectl rollout restart deployment opa -n opa
```

- Observe that bookbuyer calls are now being allowed:
```
--- bookbuyer:[ 2663 ] -----------------------------------------

Fetching http://bookstore.bookstore:14001/books-bought
Request Headers: map[Client-App:[bookbuyer] User-Agent:[Go-http-client/1.1]]
Identity: bookstore-v1
Booksbought: 1087
Server: envoy
Date: Tue, 04 May 2021 02:00:46 GMT
Status: 200 OK
MAESTRO! THIS TEST SUCCEEDED!

Fetching http://bookstore.bookstore:14001/buy-a-book/new
Request Headers: map[]
Identity: bookstore-v1
Booksbought: 1088
Server: envoy
Date: Tue, 04 May 2021 02:00:47 GMT
Status: 200 OK
ESC[90m2:00AMESC[0m ESC[32mINFESC[0m BooksCountV1=21490056 ESC[36mcomponent=ESC[0mdemo ESC[36mfile=ESC[0mbooks.go:167
MAESTRO! THIS TEST SUCCEEDED!
```

```
{"decision_id":"3f29d449-7f71-4721-b93c-ad7d375e0f80","input":{"attributes":{"destination":{"address":{"socketAddress":{"address":"10.0.16.44","portValue":14001}}},"metadataContext":{},"request":{"http":{"headers":{":authority":"bookstore.bookstore:14001",":method":"GET",":path":"/buy-a-book/new","accept-encoding":"gzip","user-agent":"Go-http-client/1.1","x-forwarded-proto":"http","x-request-id":"97bd9339-448f-4710-bba4-bda3b5103aa0"},"host":"bookstore.bookstore:14001","id":"14741973070759351541","method":"GET","path":"/buy-a-book/new","protocol":"HTTP/1.1"},"time":"2021-05-04T02:01:35.813125Z"},"source":{"address":{"socketAddress":{"address":"10.244.2.10","portValue":48860}}}},"parsed_body":null,"parsed_path":["buy-a-book","new"],"parsed_query":{},"truncated_body":false,"version":{"encoding":"protojson","ext_authz":"v3"}},"labels":{"id":"e8f143eb-9edf-425d-9210-340001993841","version":"0.28.0-envoy"},"level":"info","metrics":{"timer_rego_query_eval_ns":50500,"timer_server_handler_ns":370797},"msg":"Decision Log","path":"envoy/authz/allow","requested_by":"","result":true,"time":"2021-05-04T02:01:35Z","timestamp":"2021-05-04T02:01:35.816768454Z","type":"openpolicyagent.org/decision_logs"}
```
