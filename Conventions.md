# Conventions

This document outlines the conventions accepted and guiding the development of the Service Mesh Controller.


### Certificate vs Proxy vs Endpoint

This section establishes the logical entities and relationships between `Certificate`, `Proxy`, and `Endpoint`.

> **NOTE**: The Certificate described in this section is the one issued to the Envoy proxy, by the service mesh controller, and used exclusively for Envoy to xDS mTLS communication. This certificate is different than the one issued for services and used for service-to-service mTLS communication. For clarity we will refer to this certificate as `PCertificate` (proxy certificate).

##### Definitions
  - `PCertificate` (strictly in the context of xDS mTLS, not service-to-service):
      - is issued by the service mesh controller to each individual proxy (xDS client)
      - is a client certificate validated by the xDS server
      - has Common Name in the DNS-1123 standard with the following format: `<proxy-UUID>.<service-name>`
  - `Proxy` is the logical component (Envoy):
      - fronts a mesh service process (container or binary running on Kubernetes or a VM)
      - maintains a connection to a control plane (xDS server)
      - continuously receives configuration updates (xDS protos)
  - `Endpoint` is the tuple of an IP address and a port number of a Container or a Virtual Machine, which is hosting a proxy and a process
 
 ##### Relationships
  - The `PCertificate` is issued by the service mesh controller (SMC) either manually or automatically
  - The `Proxy` is installed manually or automatically by SMC and is provisioned with the issued `PCertificate`
  - The `Endpoint` is created by Kubernetes or the cloud vendor
    - in the automated case - SMC discovers it and triggers `PCertificate` issuance and Proxy installation
    - for manual installation - user would request a new cert for a given service from SMC
  - The **intersection** of the set of issued xDS `PCertificates` ∩ `Proxies` ∩ `Endpoints` defines the backing IPs and Ports for a given service


### Envoy proxy ID and service membership

  - Each `Proxy` is issued a unique `PCertificate`, which is dedicated to xDS mTLS communication
  - `PCertificate` has a per-proxy unique Subject CN, which identifies the `Proxy`
  - The `Proxy`'s service membership is determined by examining the CN FQDN (`<proxy-UUID>.<service-name>`), where service name is string following the second period in the CN of the `PCertificate`
  - There is one unique `PCertificate` issued to one `Proxy`, which is dedicated to one unique `Endpoint`, and all of these can belong to only one `Service`
  - A mesh `Service` however would be constructed by one or more (`PCertificate` + `Proxy` + `Endpoint`) tuples 