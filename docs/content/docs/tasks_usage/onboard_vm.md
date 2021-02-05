---
title: "Onboarding VMs"
description: "Onboarding VMs to the service mesh"
type: docs
weight: 2
---

# Onboarding VMs to the service mesh

**Status: WIP**

This document describes the steps to onboard a VM to the service mesh.

## Prerequisites

- Ubuntu VM on Azure
- The VM's network interface should have connectivity to the Kubernetes POD IP addresses. A service running on the VM that is a part of the service mesh will need to establish connectivity with other services in the service mesh and the OSM control plane.

  - On AKS: AKS cluster with advanced networking enabled - required for direct connectivity between K8s pods and services within the Azure VNET

- Envoy proxy needs to be installed on the VM prior to onboarding the VM into the service mesh.

## Onboarding steps

- Extract the Envoy bootstrap configuration by running the following command

  ```
  $ osm envoy get bootstrap-config // TBD
  ```

  Copy the output to `/etc/envoy/bootstrap.yaml` on the VM and configure Envoy to start using this bootstrap config file. See [Bootstrapping the VM with Envoy proxy](#bootstrapping-the-vm-with-envoy-proxy) section for more detailed information.

- Bootstrap Envoy on the VM with the certificates and keys required

  Retrieve and copy the certificate chain and private key required for Envoy to participate in the service mesh

  ```
  $ SERVICE_NAMESPACE=osm
  $ kubectl -n $SERVICE_NAMESPACE get secret osm.default \ -o jsonpath='{.data.root-cert\.pem}' | base64 --decode > root-cert.pem
  $ kubectl -n $SERVICE_NAMESPACE get secret osm.default \ -o jsonpath='{.data.key\.pem}' | base64 --decode > key.pem
  $ kubectl -n $SERVICE_NAMESPACE get secret osm.default \ -o jsonpath='{.data.cert-chain\.pem}' | base64 --decode > cert-chain.pem
  ```

- Bootstrap the VM to resolve the DNS name of the ADS cluster

  Add the POD IP address of the Aggregated Discovery Service (ADS) to the `/etc/hosts` file on the VM.

  For ex. if `ads.osm.svc.cluster.local` resolves to `192.168.1.10`:

  ```
  $ echo "192.168.1.10 ads.osm.svc.cluster.local" | sudo  tee -a /etc/hosts
  ```

- Start Envoy on the VM

  At this point, the VM should connect to OSM and participate in the service mesh.

### Bootstrapping the VM with Envoy proxy

#### Install and set up Envoy proxy

- Install the Envoy proxy package
  ```
  $ curl -sL https://getenvoy.io/gpg | sudo apt-key add -
  $ sudo add-apt-repository "deb [arch=amd64] https://dl.bintray.com/tetrate/getenvoy-deb $(lsb_release -cs) stable"
  $ sudo apt-get update
  $ sudo apt-get install -y getenvoy-envoy
  ```
- Verify Envoy is installed
  ```
  $ envoy --version
  ```
- Copy the Envoy boostrap configuration file `osm/config/bootstrap.yaml` to `/etc/envoy/bootstrap.yaml`
  Refer to [Envoy - Getting Started guide](https://www.envoyproxy.io/docs/envoy/latest/start/start#https://www.envoyproxy.io/docs/envoy/latest/start/start#) for setting up the bootstrap configuration.

- Add the hostname to IP address mapping for the xDS services in `/etc/hosts` file on the VM so that the envoy proxy can connect to the xDS services using their hostname specified in the bootstrap config file.

- Configure the Envoy service by creating `envoy.service` file under `/etc/systemd/system` and register it as a service

  ```
  [Unit]
  Description=Envoy

  [Service]
  ExecStart=/usr/bin/envoy -c /etc/envoy/bootstrap.yaml
  Restart=always
  RestartSec=5
  KillMode=mixed
  SyslogIdentifier=envoy
  LimitNOFILE=640000

  [Install]
  WantedBy=multi-user.target
  ```

  ```
  $ systemctl daemon-reload
  ```

- Set up the certificates required for mTLS between Envoy proxies and for Envoy proxy to OSM control plane communication

  - Copy `osm/demo/certificates/*` to `/etc/certs/` on the VM
  - Copy `osm/bin/cert.pem`, `osm/bin/key.pem` to `/etc/ssl/certs/` on the VM

- Start Envoy proxy

  ```
  $ systemctl start envoy
  ```

- Check `/var/log/syslog` if you encounter issues with Envoy

- Copy and run the bookstore app `osm/demo/bin/bookstore` on the VM

More experimentation [here](/docs/design/onboard_vm/crd/README.md)
