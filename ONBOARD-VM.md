
# Onboarding VMs to the service mesh
This document describes the steps to onboard a VM to the service mesh.

## Prerequisites
- The VM's network interface should have connectivity to the Kubernetes POD IP addresses.
A service running on the VM that is a part of the service mesh will need to establish connectivity with other services in the service mesh and the SMC control plane.

- Envoy proxy needs to be installed on the VM prior to onboarding the VM into the service mesh.


## Onboarding steps

- Extract the Envoy bootstrap configuration by running the following command
	```
	$ osm envoy get bootstrap-config // TBD
	```
	Copy the output to `/etc/envoy/bootstrap.yaml` on the VM and configure Envoy to start using this bootstrap config file.

-  Bootstrap Envoy on the VM with the certificates and keys required

	Retrieve and copy the certificate chain and private key required for Envoy to participate in the service mesh
	```
	$ SERVICE_NAMESPACE=smc
	$ kubectl -n $SERVICE_NAMESPACE get secret smc.default \ -o jsonpath='{.data.root-cert\.pem}' | base64 --decode > root-cert.pem
	$ kubectl -n $SERVICE_NAMESPACE get secret smc.default \ -o jsonpath='{.data.key\.pem}' | base64 --decode > key.pem
	$ kubectl -n $SERVICE_NAMESPACE get secret smc.default \ -o jsonpath='{.data.cert-chain\.pem}' | base64 --decode > cert-chain.pem
	```

- Bootstrap the VM to resolve the DNS name of the ADS cluster

   Add the POD IP address of the Aggregated Discovery Service (ADS) to the `/etc/hosts` file on the VM.

   For ex. if `ads.smc.svc.cluster.local` resolves to `192.168.1.10`:
   ```
  $ echo "192.168.1.10 ads.smc.svc.cluster.local" | sudo  tee -a /etc/hosts
   ```

- Start Envoy on the VM

	At this point, the VM should connect to SMC and participate in the service mesh.
