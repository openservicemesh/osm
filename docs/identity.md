- [Identity in OSM](#identity-in-osm)
- [How Identity is represented in OSM](#how-identity-is-represented-in-osm)
  - [The role of Certificates](#the-role-of-certificates)
- [Establishing a Pods Identity](#establishing-a-pods-identity)
  - [Pod Admission](#pod-admission)
    - [Envoy Sidecar](#envoy-sidecar)
      - [Confirming Pod Identity when requesting configuration from xDS](#confirming-pod-identity-when-requesting-configuration-from-xds)
    - [Service to Service communication certificates](#service-to-service-communication-certificates)
- [Identity in Action](#identity-in-action)
  - [SMI Traffic Policy](#smi-traffic-policy)
  - [Permissive Mode](#permissive-mode)

## Identity in OSM

To provide features like mTLS, traffic policies and traffic splits, a service mesh assigns each pod an Identity.

This happens in two steps.  First OSM will need to establish an Identity for
each pod. Second when the pods are communicating,  `Pod A` will need to confirm `Pod B`'s identity and vice versa. This document will go over how this is achieved in OSM.

## How Identity is represented in OSM

Internally, the [Identity](https://github.com/openservicemesh/osm/tree/main/pkg/identity) of an application is represented as a string. It is the unique string for an
application which represents enough information to be uniquely identified within a given environment. The string
format could have multiple representations depending on the environment the
application is running in such as a VM, Kubernetes, or cloud service.  Currently
the only environment that is represented in OSM is Kubernetes.

An example of an Identity in Kubernetes for an application is the following. It uses the Service Account assigned to the Pod and the namespace the pod is running in:

```
<ServiceAccount>.<Namespace>
```

The application's Identity is also bound to a trust domain. A trust domain is a logical
boundary in which the application can be trusted.  By default, OSM's trust domain is
`cluster.local`. This trust domain is encoded in the certificate signing
request (CSR) so it can be used during mTLS setup. The trust domain is configurable in OSM because some Cluster Authorities (CA's) may have policies
that require that the CSR's use the company's trust domain such as `test.company.com`.

With the trust domain included, the full application Identity (known as the `Principal Identity`) is:

```
<ServiceAccount>.<Namespace>.<trust-domain>
```

### The role of Certificates

After an [Identity is established](#establishing-a-pods-identity) two certificates are issued for the Pod. The first is used by Envoy to receive updates from the OSM control plane.  The second
certificate is the service certificate and has the Applications Identity encoded in the subject of the certificate (with the
trust domain).  When a pod (`Pod A`) communicates with
another pod (`Pod B`), `Pod A`'s Envoy instance will present its service certificate
to `Pod B`. `Pod B`'s Envoy instance will verify the
certificate was signed by a Certificate Authority it trusts and vice versa.
This use of certificates is how mTLS is established and can be used for [Traffic Policies](https://release-v1-2.docs.openservicemesh.io/docs/getting_started/traffic_policies/#traffic-policies).

> note: Although, OSM customers are primarily concerned about Pod to Pod
> communication, OSM control plane components are also issued a certificate for subsystems. 
> Some examples of uses are for communication with the Kubernetes API server and the Envoy Aggregated Discovery Service.

Learn more about [Certificate issuance and usage](certificate_management.md).

## Establishing a Pods Identity

### Pod Admission
When a pod is created, the [OSM's Mutating Webhook](https://github.com/openservicemesh/osm/blob/main/DESIGN.md#pod-lifecycle) injects Envoy as a sidecar with a
bootstrap `Proxy Certificate`. This [Proxy
Certificate](../DESIGN.md#b-proxy-tls-certificate) is used to gather information
from the [Envoy xDS server in the OSM control
plane](https://github.com/openservicemesh/osm/blob/main/DESIGN.md#1-proxy-control-plane) and allows for dynamically updating which endpoints a Proxy can talk to, enables service certificate rotations and more. Learn more about the [design of
OSM](../DESIGN.md) which goes into more details on how this works.

#### Envoy Sidecar
During the Mutating webhook step, the Pod Identity is established using the
Pod's Service account and namespace.  The Envoy `Proxy Certificate` has the Pod
Identity encoded in the Common Name (CN) field and DNS Subject Alternative Name (SAN) field of the certificate and is later
used to confirm Pod Identity and issue certificates for Pod to Pod
communication. 

The Envoy `Proxy Certificate` CN and SAN DNS name has the following format:

```
<proxy-UUID>.<kind>.<proxy-identity>.<trust-domain>
```

- `proxy-uuid` = random unique GUID
- `kind` = `sidecar` 
- `proxy-identity` = For Kubernetes this identity is from the pod spec
  in the form of `<service-account>.<namespace>`
- `trust-domain` = default is `local.cluster`

An example of this cert can be seen by examining the Envoy proxy configuration:

```bash
# from another terminal
osm proxy get config_dump bookwarehouse-78757788bb-zs4hk -n bookwarehouse | jq -r '.configs[] | select(."@type"=="type.googleapis.com/envoy.admin.v3.SecretsConfigDump") | .dynamic_active_secrets[] | select(.name=="tls_sds").secret.tls_certificate.certificate_chain.inline_bytes' | base64 -d | openssl x509 -noout -text


Certificate:
    Data:
        Version: 3 (0x2)
        Serial Number:
            e2:33:ea:fc:c1:59:75:1c:3e:be:f6:4c:8e:67:4d:ab
        Signature Algorithm: sha256WithRSAEncryption
        Issuer: C = US, L = CA, O = Open Service Mesh, CN = osm-ca.openservicemesh.io
        Validity
            Not Before: Aug 15 16:37:14 2022 GMT
            Not After : Aug 12 16:37:14 2032 GMT
        Subject: O = Open Service Mesh, CN = a00b5103-bcff-40d1-aebc-0fd88cd541a7.sidecar.bookwarehouse.bookwarehouse.cluster.local
        
        ...

        X509v3 extensions:
            ...
            X509v3 Subject Alternative Name:
                DNS:a00b5103-bcff-40d1-aebc-0fd88cd541a7.sidecar.bookwarehouse.bookwarehouse.cluster.local
```

##### Confirming Pod Identity when requesting configuration from xDS
When a proxy establishes a requests to the OSM Control plane.  The xDS server
will confirm the envoy side car has a trusted Certificate.  Additionally before
processing any requests, the xDS service will also gather Pod meta data and
confirm the Service Account in the presented Certificate matches the service
Account of the running Pod. 

#### Pod to Pod certificates

When the Proxy requests its certificate for pod to pod communication,
The certificate is encoded with Pod's Identity by using the following CN and DNS SAN:

```
<ServiceAccount>.<Namespace>.<trustdomain>
```

An example of this cert can be seen by examining the Envoy proxy configuration:

```bash
# from another terminal
osm proxy get config_dump bookwarehouse-78757788bb-zs4hk -n bookwarehouse | jq -r '.configs[] | select(."@type"=="type.googleapis.com/envoy.admin.v3.SecretsConfigDump") | .dynamic_active_secrets[] | select(.name == "service-cert:bookwarehouse/bookwarehouse").secret.tls_certificate.certificate_chain.inline_bytes' | base64 -d | openssl x509 -noout -text

Certificate:
    Data:
        Version: 3 (0x2)
        Serial Number:
            25:d1:f0:08:39:01:4d:c6:db:b7:cd:20:96:0a:c0:21
        Signature Algorithm: sha256WithRSAEncryption
        Issuer: C = US, L = CA, O = Open Service Mesh, CN = osm-ca.openservicemesh.io
        Validity
            Not Before: Aug 15 16:37:29 2022 GMT
            Not After : Aug 16 16:37:29 2022 GMT
        Subject: O = Open Service Mesh, CN = bookwarehouse.bookwarehouse.cluster.local
        
        ....
        
        X509v3 extensions:
            ...
            X509v3 Subject Alternative Name:
                DNS:bookwarehouse.bookwarehouse.cluster.local
```

This is important as it is used as to determine which services can communicate
with each other when using the [SMI Traffic Policy
Mode](https://docs.openservicemesh.io/docs/getting_started/traffic_policies/#smi-traffic-policy-mode) as seen in [Identity in Action](#identity-in-action)

## Identity in Action

Beyond ensuring that all communication is encrypted with trusted certificates
signed by the Root CA, OSM also uses the Identity to confirm that `Pod A` is allowed
to talk to `Pod B`.

### SMI Traffic Policy
When [SMI Traffic Policy
Mode](https://docs.openservicemesh.io/docs/getting_started/traffic_policies/#smi-traffic-policy-mode)
is enabled, the Identity encoded in the certificate is used to confirm that the
services that establishes a connection is allowed to communicate.  This is done
via [Envoy RBAC filters](https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/rbac/v3/rbac.proto#envoy-v3-api-msg-config-rbac-v3-principal-authenticated) that are created when the Proxy is [requesting its
service
configuration](https://github.com/openservicemesh/osm/tree/main/pkg/envoy/rbac)
from the xDS server. 

Using the book buyer demo we can turn on SMI Traffic Policy and [set up the following traffic policy](https://docs.openservicemesh.io/docs/getting_started/traffic_policies/#deploy-smi-access-control-policies) from `bookbuyer` application to `bookstore`:

```yaml
kind: TrafficTarget
apiVersion: access.smi-spec.io/v1alpha3
metadata:
  name: bookstore
  namespace: bookstore
spec:
  destination:
    kind: ServiceAccount
    name: bookstore
    namespace: bookstore
  rules:
  - kind: HTTPRouteGroup
    name: bookstore-service-routes
    matches:
    - buy-a-book
    - books-bought
  sources:
  - kind: ServiceAccount
    name: bookbuyer
    namespace: bookbuyer
---
apiVersion: specs.smi-spec.io/v1alpha4
kind: HTTPRouteGroup
metadata:
  name: bookstore-service-routes
  namespace: bookstore
spec:
  matches:
  - name: books-bought
    pathRegex: /books-bought
    methods:
    - GET
    headers:
    - "user-agent": ".*-http-client/*.*"
    - "client-app": "bookbuyer"
  - name: buy-a-book
    pathRegex: ".*a-book.*new"
    methods:
    - GET
```

After applying the TrafficTarget, we can see the following configuration on the `bookstore` Envoy configuration at the L4 layer. Only the `bookbuyer` Identity is listed as the principals allowed to connect. The Envoy RBAC filter will verify that the `bookbuyer.bookbuyer.cluster.local` identity is in the Certificate presented from the connecting Pod.

```bash
osm proxy get config_dump bookstore-65fd4c5589-wmh9k -n bookstore | jq -r '.configs[] | select(."@type"=="type.googleapis.com/envoy.admin.v3.ListenersConfigDump") | .dynamic_listeners[] | select(.name == "inbound-listener").active_state.listener.filter_chains[0].filters[0]' 

{
  "name": "l4_rbac",
  "typed_config": {
    "@type": "type.googleapis.com/envoy.extensions.filters.network.rbac.v3.RBAC",
    "rules": {
      "policies": {
        "bookstore/bookstore": {
          "permissions": [
            {
              "any": true
            }
          ],
          "principals": [
            {
              "authenticated": {
                "principal_name": {
                  "exact": "bookbuyer.bookbuyer.cluster.local"
                }
              }
            }
          ]
        }
      }
    },
    "stat_prefix": "network-"
  }
}
```

The HTTP route config also has the `bookstore` Identity. Having Idenities at this layer enables OSM to wire different Identities to specific HTTP routes on a service. In this case we've only set up the `bookbuyer`.

```bash
osm proxy get config_dump bookstore-65fd4c5589-wmh9k -n bookstore | jq -r '.configs[] | select(."@type"=="type.googleapis.com/envoy.admin.v3.RoutesConfigDump") | .dynamic_route_configs[0].route_config.virtual_hosts[0].routes'


[
  {
    "match": {
      "headers": [
        {
          "name": ":method",
          "safe_regex_match": {
            "google_re2": {},
            "regex": "GET"
          }
        }
      ],
      "safe_regex": {
        "google_re2": {},
        "regex": ".*a-book.*new"
      }
    },
    "route": {
      "weighted_clusters": {
        "clusters": [
          {
            "name": "bookstore/bookstore-v1|14001|local",
            "weight": 100
          }
        ],
        "total_weight": 100
      },
      "timeout": "0s"
    },
    "typed_per_filter_config": {
      "envoy.filters.http.rbac": {
        "@type": "type.googleapis.com/envoy.extensions.filters.http.rbac.v3.RBACPerRoute",
        "rbac": {
          "rules": {
            "policies": {
              "rbac-for-route": {
                "permissions": [
                  {
                    "any": true
                  }
                ],
                "principals": [
                  {
                    "authenticated": {
                      "principal_name": {
                        "exact": "bookbuyer.bookbuyer.cluster.local"
                      }
                    }
                  }
                ]
              }
            }
          }
        }
      }
    }
  },
```

If another Pod with a different Identity (for instance `bookthief`) was able to connect to the `bookstore` then the request would be denied, even if the certificate was signed by a trusted authority. 

We can see the Identity that the `bookbuyer` is allowed to connect to is `bookstore` as well.  Again if another pod was able to impersonate the `bookstore` but presented a different certificate then the request would fail.

```
osm proxy get config_dump bookbuyer-b8c7bc4d9-jhtmx -n bookbuyer | jq -r '.configs[] | select(."@type"=="type.googleapis.com/envoy.admin.v3.SecretsConfigDump") | .dynamic_active_secrets[] | select(.name == "root-cert-for-mtls-outbound:bookstore/bookstore-v1").secret.tls_certificate.certificate_chain.inline_bytes' | base64 -d | openssl x509 -noout -text

{
  "@type": "type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.Secret",
  "name": "root-cert-for-mtls-outbound:bookstore/bookstore-v1",
  "validation_context": {
    "trusted_ca": {
      "inline_bytes": "<redacted>"
    },
    "match_subject_alt_names": [
      {
        "exact": "bookstore-v1.bookstore.cluster.local"
      }
    ]
  }
}
```

### Permissive Mode
[Permissive
mode](https://docs.openservicemesh.io/docs/guides/traffic_management/permissive_mode/#how-it-works)
mode is configured by using wild card Traffic policies and RBAC rules are set to any `Any`. 

For instance, if we turn Permissive Mode on and review the same configuration we see:

```bash
osm proxy get config_dump bookstore-65fd4c5589-wmh9k -n bookstore | jq -r '.configs[] | select(."@type"=="type.googleapis.com/envoy.admin.v3.RoutesConfigDump") | .dynamic_route_configs[0].route_config.virtual_hosts[0].routes'

[
  {
    "match": {
      "headers": [
        {
          "name": ":method",
          "safe_regex_match": {
            "google_re2": {},
            "regex": ".*"
          }
        }
      ],
      "safe_regex": {
        "google_re2": {},
        "regex": ".*"
      }
    },
    "route": {
      "weighted_clusters": {
        "clusters": [
          {
            "name": "bookstore/bookstore-v1|14001|local",
            "weight": 100
          }
        ],
        "total_weight": 100
      },
      "timeout": "0s"
    },
    "typed_per_filter_config": {
      "envoy.filters.http.rbac": {
        "@type": "type.googleapis.com/envoy.extensions.filters.http.rbac.v3.RBACPerRoute",
        "rbac": {
          "rules": {
            "policies": {
              "rbac-for-route": {
                "permissions": [
                  {
                    "any": true
                  }
                ],
                "principals": [
                  {
                    "any": true
                  }
                ]
              }
            }
          }
        }
      }
    }
  }
]
```


