# SMC
The Service Mesh Controller (SMC) is a light weight control plane for Service Meshes. It works with Envoy proxies(and other xDS compliant data plane proxies), configured as side-car containers, and continuously programs them to implement Service Mesh policies. It provides the following key benefits
1. Native support for Virtual Machines. Can be easily extended to support Serverless workloads also. 
2. Compatible with Service Mesh Interface specification. Users can express Service Mesh policies through SMI
3. Provides declarative APIs to add and remove Kubernetes Services and VMs in a mesh. Supports Hybrid Meshes comprising of K8S services, VMs and other types of compute instances. 
4. Provides auto-injection of Envoy proxy in Kubernetes services and Virtual Machines when added to the mesh
5. Provides a pluggable interface to integrate with external certificate management services/solutions 

The SMC is made up of the following components
## SMI Monitor
Responsible for processing new SMI manifests or updates to applied manifests.

## Service Registry
The service registry maintains a catalog of all services available within the service mesh. This includes standard Kubernetes services and other proprietary service primitives such as an Azure URI representing a VM or a VM scale set. 

## Informers 
Informers continuously monitor for updates to services (such as addition or removal of endpoints in the service) and addition of new services and dispatch the configuration changes to the Envoy XDS server which pushes them down to the Envoy proxies. There are two types of informers
		○ Kubernetes Informers that register with the Kubernetes Master monitor for changes to Kubernetes Services
		○ Proprietary informers, specific to the mesh environment, that monitor VM controllers/providers for changes to VMs and VM scale sets. 
    
## Certificate Issuer 
Deals with issuance, secure storage and distribution of certificates to all registered Envoy proxies. The component acts as a CA to issue, store, sign, renew and revoke certificates for each service in a mesh. It also provides a pluggable interface for integrating with external key management solutions, such as the Azure Key Vault. It subscribes to the events of the Service Registry and reacts by creating or revoking certificates for newly added or removed services. Certificate Issuer is also responsible for notifying (passively, via event bus) the Envoy XDS for needed updates.

## Sidecar Injector
The Sidecar Injector injects Envoy proxy side car into Kubernetes services and Virtual Machines that are part of the service mesh. The component is also responsible for creating the minimum viable configuration for a service and provisioning it on to the Envoy sidecar container.

## Envoy xDS
Implements a gRPC server that acts as a control plane to program configuration updates to all deployed Envoy proxies.
