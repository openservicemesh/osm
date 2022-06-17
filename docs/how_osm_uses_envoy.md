# How OSM Uses Envoy

OSM heavily uses envoy for proxying requests, applying mTLS, and exporting statistics. Envoy itself has 5 main
configuration mechanisms for request manipulation:

* ListenerConfigs
* RouteConfigs
* EndpointConfigs
* ClusterConfigs
* SecretConfigs

## Listeners

Envoy is able to intercept all inbound and outbound traffic through [IPtables redirection](./iptables_redirection.md)

In addition to the address and port to listen on, a Listener is configured with a set of `filter_chains` which dictate
what to look for and what to do with a matching request, and a `listener_filter`, which tells Envoy what data it needs
to extract from the request.

Each listener will have a single `filter_chain` per service, per port. That is, the inbound listener will have 1 filter
chain for each port on each service that can reference its associated Pod, and the outbound filter will have 1 filter
chain for each port on each service the Pod can reach. A filter chain is comprised of 2 parts, a `filter_chain_match`,
and a list of `filters`. This gives us the following construct:

1. ListenerFilters
2. FilterChains
    1. Filter chain matches
    2. List of filters

The main listener filter used in OSM is the "Original Destination" filter which tells the listener to pick up the
address from a syscall `getsockopt` with the option `SO_ORIGINAL_DST` that gets the originally requested IP address from
the socket (prior to the IP table redirect). Otherwise, the IP used in the filter chain match would be the the
redirected IP referencing the sidecar itself.

### Example flow

The flow of a request from `pod-1` backed by `service-a` to `service-b` is as follows:

The pod performs a DNS lookup for `service-b`, which routes to the in-cluster DNS service (typically CoreDNS or KubeDNS)
which will return the *service* ClusterIP address (we are currently ignoring the case for headless services, or
ExternalName). This DNS request, which happens over UDP, is permitted and not redirected via the IP Tables redirects.

Next, the pod will attempt to make a request to that IP address, on that port. The IP tables are setup to redirect the
to port 15001, which the `outbound-listener` is listening on. The Listener is configured to extract the SO_ORIGINAL_DST,
and passes that through the set of filter chains, looking for the most precise match.

On a filter match, envoy will now apply that filter chain's full set of filters, 1 by 1, in order. The Filter we are
interested in, is called the `HttpConnectionManager`, which uses rDS to pass apply a RouteConfiguration, which OSM calls
`rds-outbound` to the request.

Below is a significantly paired down Listener configuration, which depicts the relevant info we just went over.

```json
 {
      "name":"outbound-listener",
      "active_state":{
         "listener":{
            "address":{
               "socket_address":{
                  "address":"0.0.0.0", // This tells envoy to listen on all hosts
                  "port_value":15001 // The outbound listener port. The IP Tables must intercept all outbound traffic and redirect here.
               }
            },
            "filter_chains":[
               {
                  "filter_chain_match":{
                     "prefix_ranges":[
                        {
                           "address_prefix":"10.96.75.109", // This is populated by the Service, and is what is matched by the SO_ORIGINAL_DST.
                           "prefix_len":32
                        }
                     ],
                     "destination_port":14001 // The destination port must be equal to this.
                  },
                  "filters":[
                     {
                        "name":"http_connection_manager",
                        "typed_config":{
                           "@type":"type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
                           "rds":{
                               // This tells the listener which RouteConfiguration to use.
                              "route_config_name":"rds-outbound"
                           },
                        }
                     }
                  ],
                  "name":"outbound-mesh-http-filter-chain:bookwarehouse/bookwarehouse"
               }
            ],
            "listener_filters":[
               {
                  "name":"envoy.filters.listener.original_dst"
               }
            ],
         },
      }
}
```


## Routes

The next step is the RoutesConfig. The RoutesConfig is comprised of a series of "virtual hosts", one for each service.
When the route config is applied to a request, it will iterate over the list of virtual hosts, looking for a match. A
virtual host is comprised of a set of domains it applies to, and a set of routes, where each route has some match
criteria (ie: inspecting headers, path, etc), and a set of clusters. Once a match is determined it will pick one of the
available clusters, based on a load balancing scheme (ie: weighted round robin).

Below is a significantly paired down Route configuration, which depicts the relevant info we just went over:

```json
{
   "@type":"type.googleapis.com/envoy.config.route.v3.RouteConfiguration",
   "name":"rds-outbound",
   "virtual_hosts":[
      {
         "name":"outbound_virtual-host|bookstore",
         "domains":[ // The set of domains that will match this virtual host.
            "bookstore",
            "bookstore.bookstore",
            "bookstore.bookstore.svc",
            "bookstore.bookstore.svc.cluster",
            "bookstore.bookstore.svc.cluster.local",
            "bookstore:14001",
            "bookstore.bookstore:14001",
            "bookstore.bookstore.svc:14001",
            "bookstore.bookstore.svc.cluster:14001",
            "bookstore.bookstore.svc.cluster.local:14001"
         ],
         "routes":[
            {
               "match":{ // To apply these routes, all of the matches need to be satisfied.
                  "headers":[
                     {
                        "name":":method",
                        "safe_regex_match":{
                           "regex":".*"
                        }
                     }
                  ],
               },
               "route":{
                  "weighted_clusters":{
                     "clusters":[ // There are multiple clusters when we create a TrafficSplit. Envoy will pick one of
                     of these clusters to route to.
                        {
                           "name":"bookstore/bookstore-v1",
                           "weight":50
                        },
                        {
                           "name":"bookstore/bookstore-v2",
                           "weight":50
                        }
                     ],
                     "total_weight":100
                  }
               }
            }
         ]
      },
      {
         "name":"outbound_virtual-host|bookwarehouse.bookwarehouse",
         "domains":[ // The set of domains that will match this virtual host.
            "bookwarehouse.bookwarehouse",
            "bookwarehouse.bookwarehouse.svc",
            "bookwarehouse.bookwarehouse.svc.cluster",
            "bookwarehouse.bookwarehouse.svc.cluster.local",
            "bookwarehouse.bookwarehouse.svc.cluster.local:14001",
            "bookwarehouse.bookwarehouse.svc.cluster:14001",
            "bookwarehouse.bookwarehouse.svc:14001",
            "bookwarehouse.bookwarehouse:14001"
         ],
         "routes":[
            {
               "match":{
                  "headers":[
                     {
                        "name":":method",
                        "safe_regex_match":{
                           "google_re2":{

                           },
                           "regex":".*"
                        }
                     }
                  ],
               },
               "route":{
                  "weighted_clusters":{ // Envoy will route to this cluster.
                     "clusters":[
                        {
                           "name":"bookwarehouse/bookwarehouse",
                           "weight":100
                        }
                     ],
                     "total_weight":100
                  }
               }
            }
         ]
      }
   ],
   "validate_clusters":false
},
"last_updated":"2021-06-03T16:30:49.174Z"
}

```

## Clusters

With a cluster chosen from the routes above, Envoy can now apply the Cluster Configuration. The cluster configuration
is relatively simple, which picks the server and client certs for mTLS, the SNI field for routing TLS, and the endpoints
from the endpoints discovery service. An Envoy cluster is the closest concept to a Kubernetes or OSM Service.

```json
{
   "@type":"type.googleapis.com/envoy.config.cluster.v3.Cluster",
   "name":"bookwarehouse/bookwarehouse", // The name of the cluster, which is matched against both the routes and the EDS.
   "type":"EDS",
   "eds_cluster_config":{
      "eds_config":{
         "ads":{ // This tells envoy to pick up endpoints from the Endpoint Discovery Service.
            // Could optionally set a "service_name" field here to tell Envoy to not use the cluster_name for the mapping
         },
      }
   },
   "transport_socket":{
      "name":"envoy.transport_sockets.tls",
      "typed_config":{
         "@type":"type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext",
         "common_tls_context":{
            "tls_certificate_sds_secret_configs":[
               {
                  "name":"service-cert:bookstore/bookstore-v2", // Use this secret for the service cert.
                  "sds_config":{
                     "ads":{

                     },
                     "resource_api_version":"V3"
                  }
               }
            ],
            "validation_context_sds_secret_config":{
               "name":"root-cert-for-mtls-outbound:bookwarehouse/bookwarehouse", // use the secret for mtls validation.
               "sds_config":{
                  "ads":{

                  },
                  "resource_api_version":"V3"
               }
            }
         },
         "sni":"bookwarehouse.bookwarehouse.svc.cluster.local"
      }
   },
},
```

NOTE: Envoy heavily uses these mapping concepts, typically based on cluster name, to match from one entity to another,
to allow for reuse. ie: the `rds-outbound` mapping above.

## Endpoints

The final step of the outbound flow, Envoy matches the set of endpoints based on the cluster_name, or the service_name
field set above, and routes to one of the available endpoints. For OSM, the endpoints will contain the ip + port for
every Pod backing the service.


```json
{
   "@type":"type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment",
   "cluster_name": "bookwarehouse/bookwarehouse", // Name of the cluster to match, or service_name if provided.
   "endpoints":[
      {
         "locality":{
            "zone":"zone"
         },
         "lb_endpoints":[
            {
               "endpoint":{
                  "address":{
                     "socket_address":{
                        "address":"10.244.0.21", // Pod address
                        "port_value":14001
                     }
                  },
               },
               "health_status":"HEALTHY",
               "load_balancing_weight":100
            }
         ]
      }
   ]
}
```