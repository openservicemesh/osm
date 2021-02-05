---
title: "Design Appendix"
description: "Design Appendix"
type: docs
weight: 7
---

## Appendix

### Fundamental Types

The following types are referenced in the interfaces proposed in this document:

- Port

  ```go
  // Port is a numerical port of an Envoy proxy
  type Port int
  ```

- ServiceName

  ```go
  // ServiceName is the name of a service defined via SMI
  type ServiceName string
  ```

- ServiceAccount

  ```go
  // ServiceAccount is a type for a service account
  type ServiceAccount string
  ```

- Endpoint
  ```go
  // Endpoint is a tuple of IP and Port, representing an Envoy proxy, fronting an instance of a service
  type Endpoint struct {
      net.IP `json:"ip"`
      Port   `json:"port"`
  }
  ```
- ClusterName
  ```go
  // ClusterName is a type for a service name
  type ClusterName string
  ```
- WeightedService

  ```go
  //WeightedService is a struct of a service name and its weight
  type WeightedService struct {
   ServiceName MeshService `json:"service_name:omitempty"`
   Weight      int               `json:"weight:omitempty"`
  }
  ```

- RoutePolicy

  ```go
  // RoutePolicy is a struct of a path and the allowed methods on a given route
     type RoutePolicy struct {
      PathRegex string   `json:"path_regex:omitempty"`
      Methods   []string `json:"methods:omitempty"`
     }
  ```

- WeightedCluster
  ```go
  // WeightedCluster is a struct of a cluster and is weight that is backing a service
     type WeightedCluster struct {
      ClusterName ClusterName `json:"cluster_name:omitempty"`
      Weight      int         `json:"weight:omitempty"`
     }
  ```
- TrafficResources
  ```go
  //TrafficResource is a struct of the various resources of a source/destination in the TrafficPolicy
  type TrafficResource struct {
     ServiceAccount ServiceAccount      `json:"service_account:omitempty"`
     Namespace      string              `json:"namespace:omitempty"`
     Services       []MeshService `json:"services:omitempty"`
     Clusters       []WeightedCluster   `json:"clusters:omitempty"`
    }
  ```
