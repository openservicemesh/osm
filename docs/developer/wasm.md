# WebAssembly Envoy Extensions

OSM includes a [WebAssembly extension](/wasm/stats.cc) to Envoy. It extends Envoy's statistics to enable [SMI metrics](https://github.com/servicemeshinterface/smi-metrics) and is built using the [proxy-wasm-cpp-sdk](https://github.com/proxy-wasm/proxy-wasm-cpp-sdk).

## Build
Building the WASM module requires only Docker as a pre-requisite and can be invoked as `make bin/osm-controller/stats.wasm` directly, or automatically as part of `make docker-build-osm-controller`.

## How it Works
Each proxy is configured by xDS to add HTTP headers prefixed with `osm-` with the metadata required for the [metrics' labels](/docs/content/docs/patterns/observability.md#custom-metrics) to each request and response, like workload name, namespace, etc. Because of the [order in which Envoy processes HTTP filters](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/http/http_filters#filter-ordering), response headers to add are configured by the router filter via RDS while request headers to add are configured via a Lua extension so that both sets of headers are made available to the WASM extension.

Each proxy only knows about its own metadata, so metadata about the downstream proxy is read from request headers while metadata about the upstream proxy is read from response headers.

Request durations represent the time between when stream context is created and ended/reset on the downstream proxy.
